package autorelay

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	basic "github.com/libp2p/go-libp2p/p2p/host/basic"
	relayv1 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv1/relay"
	circuitv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	circuitv2_proto "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/proto"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

const (
	protoIDv1 = string(relayv1.ProtoID)              //中继协议V1
	protoIDv2 = string(circuitv2_proto.ProtoIDv2Hop) //中继协议V2
)

// Terminology:
// Candidate: Once we connect to a node and it supports (v1 / v2) relay protocol,
// we call it a candidate, and consider using it as a relay.
// 候选者：一旦连接到支持中继器V1，V2的协议，我们将会选为候选者，考虑作为中继器
// Relay: Out of the list of candidates, we select a relay to connect to.
// Currently, we just randomly select a candidate, but we can employ more sophisticated
// selection strategies here (e.g. by facotring in the RTT).
// 中继器：从候选者列表中，我们选择一个中继器，当前我们随机选择

const (
	rsvpRefreshInterval = time.Minute     //预约服务刷新间隔
	rsvpExpirationSlack = 2 * time.Minute //过期等待时间

	autorelayTag = "autorelay" //自动中继器标签
)

// 候选者
type candidate struct {
	added           time.Time     // 添加时间
	supportsRelayV2 bool          // 是否支持v2
	ai              peer.AddrInfo // 地址
}

// relayFinder is a Host that uses relays for connectivity when a NAT is detected.
// 中继发现器为host，当NAT探测连接性中继时使用
type relayFinder struct {
	bootTime time.Time        //启动时间
	host     *basic.BasicHost //host

	conf *config

	refCount sync.WaitGroup //CountDownLock

	ctxCancel   context.CancelFunc //取消功能
	ctxCancelMx sync.Mutex         //互斥锁

	peerSource func(context.Context, int) <-chan peer.AddrInfo //源peer

	candidateFound             chan struct{}          // receives every time we find a new relay candidate 候选通道
	candidateMx                sync.Mutex             //候选互斥锁
	candidates                 map[peer.ID]*candidate //候选peerId
	backoff                    map[peer.ID]time.Time  //已使用的中继器peer，使用过后，不在重复添加
	maybeConnectToRelayTrigger chan struct{}          // cap: 1
	// Any time _something_ hapens that might cause us to need new candidates. 任何时间，一些事情发生，可以需要新的候选者
	// This could be
	// * the disconnection of a relay 中继器失联
	// * the failed attempt to obtain a reservation with a current candidate 从当前候选者中尝试获取reservation预约服务失败；
	// * a candidate is deleted due to its age 由于age过期，候选者删除
	maybeRequestNewCandidates chan struct{} // cap: 1.

	relayUpdated chan struct{} //

	relayMx sync.Mutex
	relays  map[peer.ID]*circuitv2.Reservation // rsvp will be nil if it is a v1 relay 中继服务

	cachedAddrs       []ma.Multiaddr //多播地址
	cachedAddrsExpiry time.Time      //???
}

// 创建中继器发现者
func newRelayFinder(host *basic.BasicHost, peerSource func(context.Context, int) <-chan peer.AddrInfo, conf *config) *relayFinder {
	if peerSource == nil { //中继器source不能为空
		panic("Can not create a new relayFinder. Need a Peer Source fn or a list of static relays. Refer to the documentation around `libp2p.EnableAutoRelay`")
	}

	return &relayFinder{
		bootTime:                   conf.clock.Now(),
		host:                       host, //当前host
		conf:                       conf,
		peerSource:                 peerSource,                   //源peer
		candidates:                 make(map[peer.ID]*candidate), //候选者
		backoff:                    make(map[peer.ID]time.Time),  //剔除的候选者
		candidateFound:             make(chan struct{}, 1),
		maybeConnectToRelayTrigger: make(chan struct{}, 1),
		maybeRequestNewCandidates:  make(chan struct{}, 1),
		relays:                     make(map[peer.ID]*circuitv2.Reservation), //中继器预约信息；
		relayUpdated:               make(chan struct{}, 1),
	}
}

// auto中继器
// 1. 从当前网络环境上下文中，知道对应的可以支持中继协议V2的peer节点
// 2.
func (rf *relayFinder) background(ctx context.Context) {
	rf.refCount.Add(1)
	go func() {
		defer rf.refCount.Done() //退出countdown
		//从当前网络环境上下文中，知道对应的可以支持中继协议V2的peer节点
		rf.findNodes(ctx) //节点可达状态发现
	}()

	rf.refCount.Add(1) //countdownlock 自增
	go func() {
		defer rf.refCount.Done()
		// 当前新的中继器节点发现时，maybeConnectToRelayTrigger通道将会触发一个通知
		// 新的可用中继节点产生时,将会从当前发现器中候选者中，选择支持中继V2协议的中继节点，
		// 并建立连接，同时预约中继slot。
		rf.handleNewCandidates(ctx) //
	}()
	//订阅连接状态变更，节点与其他节点联通
	subConnectedness, err := rf.host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		log.Error("failed to subscribe to the EvtPeerConnectednessChanged")
		return
	}
	defer subConnectedness.Close() //退出关闭通道连接

	bootDelayTimer := rf.conf.clock.Timer(rf.conf.bootDelay)
	defer bootDelayTimer.Stop()
	refreshTicker := rf.conf.clock.Ticker(rsvpRefreshInterval)
	defer refreshTicker.Stop()
	backoffTicker := rf.conf.clock.Ticker(rf.conf.backoff / 5)
	defer backoffTicker.Stop()
	oldCandidateTicker := rf.conf.clock.Ticker(rf.conf.maxCandidateAge / 5)
	defer oldCandidateTicker.Stop()

	for {
		// when true, we need to identify push 身份已经推送，则为true；
		var push bool

		select {
		case ev, ok := <-subConnectedness.Out():
			if !ok {
				return
			}
			evt := ev.(event.EvtPeerConnectednessChanged) //连接状态变更事件
			if evt.Connectedness != network.NotConnected {
				continue
			}
			rf.relayMx.Lock()
			if rf.usingRelay(evt.Peer) { // we were disconnected from a relay 先前使用过的中继
				log.Debugw("disconnected from relay", "id", evt.Peer)
				delete(rf.relays, evt.Peer)
				rf.notifyMaybeConnectToRelay()    //通知需要连接中继
				rf.notifyMaybeNeedNewCandidates() //通知需要新的候选者中继peer
				push = true
			}
			rf.relayMx.Unlock()
		case <-rf.candidateFound: //候选者发现，则连接中继节点
			rf.notifyMaybeConnectToRelay()
		case <-bootDelayTimer.C:
			//通知连接中继
			rf.notifyMaybeConnectToRelay()
		case <-rf.relayUpdated: //中继更新
			push = true
		case now := <-refreshTicker.C:
			// 刷线中继器的预约信息，保持连接可用，不可用的话，预约失败，在取消连接保护；
			push = rf.refreshReservations(ctx, now)
		case now := <-backoffTicker.C:
			rf.candidateMx.Lock()
			for id, t := range rf.backoff { //
				if !t.Add(rf.conf.backoff).After(now) {
					log.Debugw("removing backoff for node", "id", id)
					delete(rf.backoff, id)
				}
			}
			rf.candidateMx.Unlock()
		case now := <-oldCandidateTicker.C: //移除过期的候选者
			var deleted bool
			rf.candidateMx.Lock()
			for id, cand := range rf.candidates { //
				if !cand.added.Add(rf.conf.maxCandidateAge).After(now) {
					deleted = true
					log.Debugw("deleting candidate due to age", "id", id)
					delete(rf.candidates, id)
				}
			}
			rf.candidateMx.Unlock()
			if deleted {
				rf.notifyMaybeNeedNewCandidates() //通知需要信息的候选者
			}
		case <-ctx.Done():
			return
		}

		if push {
			rf.relayMx.Lock()
			rf.cachedAddrs = nil //缓存地址
			rf.relayMx.Unlock()
			rf.host.SignalAddressChange()
		}
	}
}

// 从当前网络环境上下文中，知道对应的可以支持中继协议V2的peer节点
// findNodes accepts nodes from the channel and tests if they support relaying. 从通道发现接受可以与节点联通的节点，并测试他们是否支持中继器
// It is run on both public and private nodes. 在公开和私钥的节点上运行
// It garbage collects old entries, so that nodes doesn't overflow. 将会回收老的entry，以防止节点溢出
// This makes sure that as soon as we need to find relay candidates, we have them available. 只要确认发现了我们需要的中继，就可以使用他们
func (rf *relayFinder) findNodes(ctx context.Context) {
	//从上下获取peer通道源，config中配置PeerSource
	peerChan := rf.peerSource(ctx, rf.conf.maxCandidates)
	var wg sync.WaitGroup //通道地址CountDownLatch
	//上次调用时间
	lastCallToPeerSource := rf.conf.clock.Now()
	//创建定时器
	timer := newTimer(rf.conf.clock)
	for {
		rf.candidateMx.Lock() //获取候选节点信息
		numCandidates := len(rf.candidates)
		rf.candidateMx.Unlock()

		if peerChan == nil {
			now := rf.conf.clock.Now()
			nextAllowedCallToPeerSource := lastCallToPeerSource.Add(rf.conf.minInterval).Sub(now)
			if numCandidates < rf.conf.minCandidates { //候选节点数据不足，则允许重启定时器
				log.Debugw("not enough candidates. Resetting timer", "num", numCandidates, "desired", rf.conf.minCandidates)
				timer.Reset(nextAllowedCallToPeerSource)
			}
		}

		select {
		case <-rf.maybeRequestNewCandidates: //需要重新筛选候选者节点
			continue
		case now := <-timer.Chan():
			timer.SetRead() //读取操作
			if peerChan != nil {
				// We're still reading peers from the peerChan. No need to query for more peers now.
				continue
			}
			lastCallToPeerSource = now
			//获取候选者peer节点
			peerChan = rf.peerSource(ctx, rf.conf.maxCandidates)
		case pi, ok := <-peerChan: //新的peer通道建立
			if !ok {
				wg.Wait() // 通道建立失败等待
				peerChan = nil
				continue
			}
			log.Debugw("found node", "id", pi.ID) //发现候选者
			rf.candidateMx.Lock()
			numCandidates := len(rf.candidates)
			backoffStart, isOnBackoff := rf.backoff[pi.ID] //判断你是否为不可用的中继器节点
			rf.candidateMx.Unlock()
			if isOnBackoff { //为不可用的中继器节点
				log.Debugw("skipping node that we recently failed to obtain a reservation with", "id", pi.ID, "last attempt", rf.conf.clock.Since(backoffStart))
				continue
			}
			if numCandidates >= rf.conf.maxCandidates { //候选者已经足够，则跳过；
				log.Debugw("skipping node. Already have enough candidates", "id", pi.ID, "num", numCandidates, "max", rf.conf.maxCandidates)
				continue
			}
			rf.refCount.Add(1)
			wg.Add(1) //WaitGroup同步器自增
			go func() {
				defer rf.refCount.Done()
				defer wg.Done()
				//如果peer节点为支持中继协议V2的节点，则添加到中继器候选者
				if added := rf.handleNewNode(ctx, pi); added {
					rf.notifyNewCandidate() //通知新的中继器候选者产生
				}
			}()
		case <-ctx.Done():
			return
		}
	}
}

func (rf *relayFinder) notifyMaybeConnectToRelay() {
	select {
	case rf.maybeConnectToRelayTrigger <- struct{}{}:
	default:
	}
}

// 通知需要新的候选者
func (rf *relayFinder) notifyMaybeNeedNewCandidates() {
	select {
	case rf.maybeRequestNewCandidates <- struct{}{}:
	default:
	}
}

// 通知新的中继器候选者产生消息
func (rf *relayFinder) notifyNewCandidate() {
	select {
	case rf.candidateFound <- struct{}{}:
	default:
	}
}

// https://github.com/libp2p/specs/blob/master/relay/circuit-v1.md
// https://github.com/libp2p/specs/blob/master/relay/README.md
// handleNewNode tests if a peer supports circuit v1 or v2. 如果peer节点支持中继circuit v1 or v2，则处理新节点
// This method is only run on private nodes. 此方法跑在私钥节点上
// If a peer does, it is added to the candidates map. 如果peer支持则，将会添加到到候选Map中
// Note that just supporting the protocol doesn't guarantee that we can also obtain a reservation. 只判断是否支持协议，不保证可以获取预约保留服务
func (rf *relayFinder) handleNewNode(ctx context.Context, pi peer.AddrInfo) (added bool) {
	rf.relayMx.Lock()
	relayInUse := rf.usingRelay(pi.ID)
	rf.relayMx.Unlock()
	if relayInUse { //已经存在响应的中继节点
		return false
	}
	//20超时context
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	//是否支持中继器V2协议
	supportsV2, err := rf.tryNode(ctx, pi)
	if err != nil {
		log.Debugf("node %s not accepted as a candidate: %s", pi.ID, err)
		return false
	}
	rf.candidateMx.Lock()
	if len(rf.candidates) > rf.conf.maxCandidates { //候选者足够
		rf.candidateMx.Unlock()
		return false
	}
	log.Debugw("node supports relay protocol", "peer", pi.ID, "supports circuit v2", supportsV2)
	//添加候选者到中继器发现器
	rf.candidates[pi.ID] = &candidate{
		added:           rf.conf.clock.Now(),
		ai:              pi,
		supportsRelayV2: supportsV2,
	}
	rf.candidateMx.Unlock()
	return true
}

// tryNode checks if a peer actually supports either circuit v1 or circuit v2. 检查节点是否支持中继协议 circuit v1 or circuit v2
// It does not modify any internal state.
func (rf *relayFinder) tryNode(ctx context.Context, pi peer.AddrInfo) (supportsRelayV2 bool, err error) {
	if err := rf.host.Connect(ctx, pi); err != nil { //建立到中继的连接
		return false, fmt.Errorf("error connecting to relay %s: %w", pi.ID, err)
	}
	//peer到中继的连接
	conns := rf.host.Network().ConnsToPeer(pi.ID)
	for _, conn := range conns {
		if isRelayAddr(conn.RemoteMultiaddr()) { //判断是否为中继地址
			return false, errors.New("not a public node")
		}
	}

	// wait for identify to complete in at least one conn so that we can check the supported protocols
	// 等待identify服务完成，确定完成至少一个连接，以便确认是否支持中继协议
	ready := make(chan struct{}, 1)
	for _, conn := range conns {
		go func(conn network.Conn) {
			select {
			case <-rf.host.IDService().IdentifyWait(conn): //等待Identify协议完成
				select {
				case ready <- struct{}{}:
				default:
				}
			case <-ctx.Done():
			}
		}(conn)
	}

	select {
	case <-ready:
	case <-ctx.Done():
		return false, ctx.Err()
	}
	//判断支持中继协议V1，V2
	protos, err := rf.host.Peerstore().SupportsProtocols(pi.ID, protoIDv1, protoIDv2)
	if err != nil {
		return false, fmt.Errorf("error checking relay protocol support for peer %s: %w", pi.ID, err)
	}

	// If the node speaks both, prefer circuit v2
	var maybeSupportsV1, supportsV2 bool
	for _, proto := range protos {
		switch proto {
		case protoIDv1:
			maybeSupportsV1 = true
		case protoIDv2:
			supportsV2 = true
		}
	}

	if supportsV2 { //支持总计V2版本
		return true, nil
	}

	if !rf.conf.enableCircuitV1 && !supportsV2 { //不支持中继
		return false, errors.New("doesn't speak circuit v2")
	}
	if !maybeSupportsV1 && !supportsV2 {
		return false, errors.New("doesn't speak circuit v1 or v2")
	}

	// The node *may* support circuit v1. 是否支持中继器V1
	supportsV1, err := relayv1.CanHop(ctx, rf.host, pi.ID)
	if err != nil {
		return false, fmt.Errorf("CanHop failed: %w", err)
	}
	if !supportsV1 {
		return false, errors.New("doesn't speak circuit v1 or v2")
	}
	return false, nil
}

// When a new node that could be a relay is found, we receive a notification on the maybeConnectToRelayTrigger chan.
// This function makes sure that we only run one instance of maybeConnectToRelay at once, and buffers
// exactly one more trigger event to run maybeConnectToRelay.
// 当前新的中继器节点发现时，maybeConnectToRelayTrigger通道将会触发一个通知
// 新的可用中继节点产生，时将会从当前发现器中候选者中，选择支持中继V2协议的中继节点，
// 并建立连接，同时预约中继slot。
func (rf *relayFinder) handleNewCandidates(ctx context.Context) {
	sem := make(chan struct{}, 1)
	for {
		select {
		case <-ctx.Done():
			return
		case <-rf.maybeConnectToRelayTrigger:
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			rf.maybeConnectToRelay(ctx)
			<-sem
		}
	}
}

// 新的可用中继节点产生时,将会从当前发现器中候选者中，选择支持中继V2协议的中继节点，
// 并建立连接，同时预约中继slot。
func (rf *relayFinder) maybeConnectToRelay(ctx context.Context) {
	rf.relayMx.Lock()
	numRelays := len(rf.relays)
	rf.relayMx.Unlock()
	// We're already connected to our desired number of relays. Nothing to do here.
	if numRelays == rf.conf.desiredRelays { //中继器足够
		return
	}

	rf.candidateMx.Lock()
	if len(rf.relays) == 0 && len(rf.candidates) < rf.conf.minCandidates && rf.conf.clock.Since(rf.bootTime) < rf.conf.bootDelay { //还在启动中
		// During the startup phase, we don't want to connect to the first candidate that we find.
		// Instead, we wait until we've found at least minCandidates, and then select the best of those.
		// However, if that takes too long (longer than bootDelay), we still go ahead.
		// 在启动阶段，我们不想连接我们连接的候选者。
		// 在我们发现最小候选者时，我们将会从minCandidates候选者中，选择最优的候选者，如果时间，超过bootDelay，则随它去
		rf.candidateMx.Unlock()
		return
	}
	if len(rf.candidates) == 0 {
		rf.candidateMx.Unlock()
		return
	}
	//获取中继发现中的候选者节点
	candidates := rf.selectCandidates()
	rf.candidateMx.Unlock()

	// We now iterate over the candidates, attempting (sequentially) to get reservations with them, until
	// we reach the desired number of relays.
	// 遍历候选者，尝试获取预约服务，知道到达需要的中继数量
	for _, cand := range candidates {
		id := cand.ai.ID
		rf.relayMx.Lock()
		usingRelay := rf.usingRelay(id)
		rf.relayMx.Unlock()
		if usingRelay { //已经在用的中继
			rf.candidateMx.Lock()
			delete(rf.candidates, id) //从发现器的候选者中删除
			rf.candidateMx.Unlock()
			//通知需要新的候选者
			rf.notifyMaybeNeedNewCandidates()
			continue
		}
		//连接中继器，并返回中继器预约信息，包括地址，有效性，凭证等
		rsvp, err := rf.connectToRelay(ctx, cand)
		if err != nil {
			log.Debugw("failed to connect to relay", "peer", id, "error", err)
			rf.notifyMaybeNeedNewCandidates() //通知需要新的候选者
			continue
		}
		log.Debugw("adding new relay", "id", id)
		rf.relayMx.Lock()
		rf.relays[id] = rsvp
		numRelays := len(rf.relays)
		rf.relayMx.Unlock()
		rf.notifyMaybeNeedNewCandidates() //通知需要新的候选者

		rf.host.ConnManager().Protect(id, autorelayTag) // protect the connection 保护连接

		select {
		case rf.relayUpdated <- struct{}{}:
		default:
		}

		if numRelays >= rf.conf.desiredRelays {
			break
		}
	}
}

// 连接中继器，并返回中继器预约信息，包括地址，有效性，凭证等
func (rf *relayFinder) connectToRelay(ctx context.Context, cand *candidate) (*circuitv2.Reservation, error) {
	id := cand.ai.ID
	//10s超时context
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var rsvp *circuitv2.Reservation

	// make sure we're still connected.
	if rf.host.Network().Connectedness(id) != network.Connected { //确保连接建立
		if err := rf.host.Connect(ctx, cand.ai); err != nil {
			rf.candidateMx.Lock()
			delete(rf.candidates, cand.ai.ID) //连接成功，从候选者删除
			rf.candidateMx.Unlock()
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	rf.candidateMx.Lock()
	rf.backoff[id] = rf.conf.clock.Now() //添加已使用的候选者
	rf.candidateMx.Unlock()
	var err error
	if cand.supportsRelayV2 { //支持V2协议
		//获取预约信息，包括地址，有效性，凭证等
		rsvp, err = circuitv2.Reserve(ctx, rf.host, cand.ai)
		if err != nil {
			err = fmt.Errorf("failed to reserve slot: %w", err)
		}
	}
	rf.candidateMx.Lock()
	delete(rf.candidates, id) //从候选者删除
	rf.candidateMx.Unlock()
	return rsvp, err
}

// 刷线中继器的预约信息，保持连接可用，不可用的话，预约失败，在取消连接保护；
func (rf *relayFinder) refreshReservations(ctx context.Context, now time.Time) bool {
	rf.relayMx.Lock()

	// find reservations about to expire and refresh them in parallel
	g := new(errgroup.Group)
	for p, rsvp := range rf.relays {
		if rsvp == nil { // this is a circuit v1 relay, there is no reservation
			continue
		}
		if now.Add(rsvpExpirationSlack).Before(rsvp.Expiration) {
			continue
		}

		p := p
		g.Go(func() error { return rf.refreshRelayReservation(ctx, p) })
	}
	rf.relayMx.Unlock()

	err := g.Wait()
	return err != nil
}

// 建立到中继器peerId的预约信息；
func (rf *relayFinder) refreshRelayReservation(ctx context.Context, p peer.ID) error {
	//获取预约信息
	rsvp, err := circuitv2.Reserve(ctx, rf.host, peer.AddrInfo{ID: p})

	rf.relayMx.Lock()
	defer rf.relayMx.Unlock()

	if err != nil {
		log.Debugw("failed to refresh relay slot reservation", "relay", p, "error", err)

		delete(rf.relays, p)
		// unprotect the connection， 取消连接保护标志
		rf.host.ConnManager().Unprotect(p, autorelayTag)
		return err
	}

	log.Debugw("refreshed relay slot reservation", "relay", p)
	//添加中继器预约信息到中继器发现器
	rf.relays[p] = rsvp
	return nil
}

// usingRelay returns if we're currently using the given relay. 当前的中继peer是否在使用
func (rf *relayFinder) usingRelay(p peer.ID) bool {
	_, ok := rf.relays[p]
	return ok
}

// selectCandidates returns an ordered slice of relay candidates.
// Callers should attempt to obtain reservations with the candidates in this order.
// 返回中继候选者，尝试着应该按顺序获取中继器预约服务；
func (rf *relayFinder) selectCandidates() []*candidate {
	candidates := make([]*candidate, 0, len(rf.candidates))
	for _, cand := range rf.candidates {
		candidates = append(candidates, cand)
	}

	// TODO: better relay selection strategy; this just selects random relays,
	// but we should probably use ping latency as the selection metric
	//随机算法
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	return candidates
}

// This function is computes the NATed relay addrs when our status is private:
//   - The public addrs are removed from the address set.
//   - The non-public addrs are included verbatim so that peers behind the same NAT/firewall
//     can still dial us directly.
//   - On top of those, we add the relay-specific addrs for the relays to which we are
//     connected. For each non-private relay addr, we encapsulate the p2p-circuit addr
//     through which we can be dialed.
func (rf *relayFinder) relayAddrs(addrs []ma.Multiaddr) []ma.Multiaddr {
	rf.relayMx.Lock()
	defer rf.relayMx.Unlock()

	if rf.cachedAddrs != nil && rf.conf.clock.Now().Before(rf.cachedAddrsExpiry) {
		return rf.cachedAddrs
	}

	raddrs := make([]ma.Multiaddr, 0, 4*len(rf.relays)+4)

	// only keep private addrs from the original addr set
	for _, addr := range addrs {
		if manet.IsPrivateAddr(addr) {
			raddrs = append(raddrs, addr)
		}
	}

	// add relay specific addrs to the list
	for p := range rf.relays {
		addrs := cleanupAddressSet(rf.host.Peerstore().Addrs(p))

		circuit := ma.StringCast(fmt.Sprintf("/p2p/%s/p2p-circuit", p.Pretty()))
		for _, addr := range addrs {
			pub := addr.Encapsulate(circuit)
			raddrs = append(raddrs, pub)
		}
	}

	rf.cachedAddrs = raddrs
	rf.cachedAddrsExpiry = rf.conf.clock.Now().Add(30 * time.Second)

	return raddrs
}

func (rf *relayFinder) Start() error {
	rf.ctxCancelMx.Lock()
	defer rf.ctxCancelMx.Unlock()
	if rf.ctxCancel != nil {
		return errors.New("relayFinder already running")
	}
	log.Debug("starting relay finder")
	ctx, cancel := context.WithCancel(context.Background())
	rf.ctxCancel = cancel
	rf.refCount.Add(1)
	go func() {
		defer rf.refCount.Done()
		rf.background(ctx)
	}()
	return nil
}

func (rf *relayFinder) Stop() error {
	rf.ctxCancelMx.Lock()
	defer rf.ctxCancelMx.Unlock()
	log.Debug("stopping relay finder")
	if rf.ctxCancel != nil {
		rf.ctxCancel()
	}
	rf.refCount.Wait()
	rf.ctxCancel = nil
	return nil
}
