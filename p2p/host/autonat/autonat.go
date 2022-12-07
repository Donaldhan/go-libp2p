package autonat

import (
	"context"
	"errors"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/eventbus"

	logging "github.com/ipfs/go-log/v2"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

var log = logging.Logger("autonat")

// AmbientAutoNAT is the implementation of ambient NAT autodiscovery
// AmbientAutoNAT自动探测
type AmbientAutoNAT struct {
	host host.Host

	*config

	ctx               context.Context
	ctxCancel         context.CancelFunc // is closed when Close is called 关闭host连接时，调用
	backgroundRunning chan struct{}      // is closed when the background go routine exits 当后台协程挂退出时，关闭

	inboundConn  chan network.Conn  // inboundConn
	observations chan autoNATResult // 自动NAT结果
	// status is an autoNATResult reflecting current status. autoNATResult反应当前的状态
	status atomic.Value
	// Reflects the confidence on of the NATStatus being private, as a single
	// dialback may fail for reasons unrelated to NAT. 反应NATStatus可以为私有的信心指数，k由于没有关联NAT的原因，单个dialbac可能失败
	// If it is <3, then multiple autoNAT peers may be contacted for dialback 如果confidence大约3，多个autoNAT peer，也许会dialback
	// If only a single autoNAT peer is known, then the confidence increases
	// for each failure until it reaches 3. 如果仅有一个autoNAT，将会尝试，知道达到3个autoNAT
	confidence   int
	lastInbound  time.Time             // 上次公网连接Inbound时间
	lastProbeTry time.Time             // 上次尝试探测时间
	lastProbe    time.Time             //上次探测时间
	recentProbes map[peer.ID]time.Time // 最近探测Peer时间

	service *autoNATService //自动探测服务

	emitReachabilityChanged event.Emitter      //可达事件发射器
	subscriber              event.Subscription //订阅器
}

// StaticAutoNAT is a simple AutoNAT implementation when a single NAT status is desired.
// 静态NAT探测
type StaticAutoNAT struct {
	host         host.Host
	reachability network.Reachability
	service      *autoNATService
}

// 自动NAT结果
type autoNATResult struct {
	network.Reachability              //网络可达性
	address              ma.Multiaddr //多播地址
}

// New creates a new NAT autodiscovery system attached to a host
// 创建一个host的 NAT 自动发现系统
func New(h host.Host, options ...Option) (AutoNAT, error) {
	var err error
	conf := new(config)
	//配置host，拨号策略
	conf.host = h
	conf.dialPolicy.host = h

	if err = defaults(conf); err != nil {
		return nil, err
	}
	if conf.addressFunc == nil { //取host地址方法
		conf.addressFunc = h.Addrs
	}

	for _, o := range options {
		if err = o(conf); err != nil {
			return nil, err
		}
	}
	//发出本地peer可达性变更事件
	emitReachabilityChanged, _ := h.EventBus().Emitter(new(event.EvtLocalReachabilityChanged), eventbus.Stateful)

	var service *autoNATService
	if (!conf.forceReachability || conf.reachability == network.ReachabilityPublic) && conf.dialer != nil { //拨号者不为空，且网络公开可达，非强制可达
		// 创建NewAutoNATService
		service, err = newAutoNATService(conf)
		if err != nil {
			return nil, err
		}
		//设置NAT协议流处理器，同时启动后台清理NAT服务请求计数器协程
		service.Enable()
	}

	if conf.forceReachability {
		//强制可达，则emit 本地可达变化事件
		emitReachabilityChanged.Emit(event.EvtLocalReachabilityChanged{Reachability: conf.reachability})

		return &StaticAutoNAT{
			host:         h,
			reachability: conf.reachability,
			service:      service,
		}, nil
	}
	//创建可取消的上线文
	ctx, cancel := context.WithCancel(context.Background())
	//创建自动探测服务
	as := &AmbientAutoNAT{
		ctx:               ctx,
		ctxCancel:         cancel,
		backgroundRunning: make(chan struct{}),
		host:              h,
		config:            conf,
		inboundConn:       make(chan network.Conn, 5), //可接受5个连接
		observations:      make(chan autoNATResult, 1),

		emitReachabilityChanged: emitReachabilityChanged,
		service:                 service,
		recentProbes:            make(map[peer.ID]time.Time), //最近探测时间
	}
	as.status.Store(autoNATResult{network.ReachabilityUnknown, nil})
	//订阅地址变更，身份round完成事件
	subscriber, err := as.host.EventBus().Subscribe([]interface{}{new(event.EvtLocalAddressesUpdated), new(event.EvtPeerIdentificationCompleted)})
	if err != nil {
		return nil, err
	}
	as.subscriber = subscriber
	//通知网络注册AutoNAT服务
	h.Network().Notify(as)
	go as.background()

	return as, nil
}

// Status returns the AutoNAT observed reachability status.
func (as *AmbientAutoNAT) Status() network.Reachability {
	s := as.status.Load().(autoNATResult)
	return s.Reachability
}

// 发出节点网络变更事件
func (as *AmbientAutoNAT) emitStatus() {
	status := as.status.Load().(autoNATResult)
	as.emitReachabilityChanged.Emit(event.EvtLocalReachabilityChanged{Reachability: status.Reachability})
}

// PublicAddr returns the publicly connectable Multiaddr of this node if one is known.
func (as *AmbientAutoNAT) PublicAddr() (ma.Multiaddr, error) {
	s := as.status.Load().(autoNATResult)
	if s.Reachability != network.ReachabilityPublic {
		return nil, errors.New("NAT status is not public")
	}

	return s.address, nil
}

func ipInList(candidate ma.Multiaddr, list []ma.Multiaddr) bool {
	candidateIP, _ := manet.ToIP(candidate)
	for _, i := range list {
		if ip, err := manet.ToIP(i); err == nil && ip.Equal(candidateIP) {
			return true
		}
	}
	return false
}

// 后台协程：
// 1. 探测定时任务触发：尝试探测tryProbe，发送DialBack探测peer，如果成功则网络可达，失败则不可达，默认为未知可达状态
// 2. 基于探测结果更新当前状态recordObservation
// 3. 订阅事件产生：peer 身份阶段完成,  尝试探测，发送DialBack探测peer，如果成功则网络可达，失败则不可达，默认为未知可达状态
// 4. 连接状态变更，连接的远端地址为公网地址，更新公网连接Inbound时间
func (as *AmbientAutoNAT) background() {
	defer close(as.backgroundRunning) //退出，关闭
	// wait a bit for the node to come online and establish some connections
	// before starting autodetection 在开始探测前，等待一会，以便节点上线，建立一些连接
	delay := as.config.bootDelay

	var lastAddrUpdated time.Time //上次地址更新时间
	//事件订阅通道
	subChan := as.subscriber.Out()
	defer as.subscriber.Close()
	defer as.emitReachabilityChanged.Close()

	timer := time.NewTimer(delay) //探测定时钟
	defer timer.Stop()
	timerRunning := true
	for {
		select {
		// new inbound connection.
		case conn := <-as.inboundConn: //有连接建立
			localAddrs := as.host.Addrs() //本地地址
			ca := as.status.Load().(autoNATResult)
			if ca.address != nil {
				localAddrs = append(localAddrs, ca.address)
			}
			if manet.IsPublicAddr(conn.RemoteMultiaddr()) &&
				!ipInList(conn.RemoteMultiaddr(), localAddrs) { // 连接的远端地址为公网地址
				as.lastInbound = time.Now() // 更新公网连接Inbound时间
			}

		case e := <-subChan: //订阅事件
			switch e := e.(type) {
			case event.EvtLocalAddressesUpdated: //本地地址更新时间
				if !lastAddrUpdated.Add(time.Second).After(time.Now()) {
					lastAddrUpdated = time.Now() //上次地址更新时间
					if as.confidence > 1 {
						as.confidence-- //信心指数减1
					}
				}
			case event.EvtPeerIdentificationCompleted: // peer 身份阶段完成
				//如果host支持的autoNet协议
				if s, err := as.host.Peerstore().SupportsProtocols(e.Peer, AutoNATProto); err == nil && len(s) > 0 {
					currentStatus := as.status.Load().(autoNATResult)
					if currentStatus.Reachability == network.ReachabilityUnknown { //且当前状态不可达，则尝试探测
						as.tryProbe(e.Peer) //尝试探测，发送DialBack探测peer，如果成功则网络可达，失败则不可达，默认为位置可达状态
					}
				}
			default:
				log.Errorf("unknown event type: %T", e)
			}

		// probe finished.
		case result, ok := <-as.observations: //探测完成autoNATResult
			if !ok {
				return
			}
			// 基于探测结果更新当前状态，
			// 1. 从不可达，变成可达ReachabilityPublic，开启autonat服务confidence置位为0； 如果先前状态为可达或者未知，在confidence小于3的情况下，自增；
			// 2. 从可达到不可达ReachabilityPrivate，如果先前为可达状态，在confidence大于0的情况下，减少confidence，如果confidence小于0，表示到其他节点也不同，关闭自动探测服务，confidence置位0；
			// 如果先前状态为非可达状态，如果confidence小于3，自增confidence；
			// 3. 如果为未知状态，，在confidence大于0的情况下，减少confidence，在先前状态为ReachabilityPrivate情况下，重启开启autonet服务；
			as.recordObservation(result)
		case <-timer.C: //尝试探测
			peer := as.getPeerToProbe() // 从peer的探测peer中选择一个探测节点
			as.tryProbe(peer)           //尝试探测，发送DialBack探测peer，如果成功则网络可达，失败则不可达，默认为未知可达状态
			timerRunning = false
		case <-as.ctx.Done():
			return
		}

		// Drain the timer channel if it hasn't fired in preparation for Resetting it.
		if timerRunning && !timer.Stop() {
			<-timer.C
		}
		//重置定时器
		// 计算下次探测调度时间
		// 1. 如果为未知状态，或者低confidence，应该放弃AutoNATRetryInterval
		// 2. 当网络可达，应该减少尝试
		// 3. 最近的连接不可公网可达，我们应该积极的尝试
		timer.Reset(as.scheduleProbe())
		timerRunning = true
	}
}

// 清除最近探测的大于退避时间的peer
func (as *AmbientAutoNAT) cleanupRecentProbes() {
	fixedNow := time.Now()
	for k, v := range as.recentProbes {
		if fixedNow.Sub(v) > as.throttlePeerPeriod {
			delete(as.recentProbes, k)
		}
	}
}

// scheduleProbe calculates when the next probe should be scheduled for.
// 计算下次探测调度时间
// 1. 如果网络状为未知状态，或者低confidence（<3），使用默认的retryInterval
// 2. 如果当网络状态为公网可达，应该减少尝试，下次调度时间增大（retryIntervalt * 2）
// 3. 如果网络状态为的公网不可达，我们应该积极的尝试（retryIntervalt / 2）
func (as *AmbientAutoNAT) scheduleProbe() time.Duration {
	// Our baseline is a probe every 'AutoNATRefreshInterval' 探测基准为AutoNATRefreshInterval间隔
	// This is modulated by:
	// * if we are in an unknown state, or have low confidence, that should drop to 'AutoNATRetryInterval'
	// 如果为未知状态，或者低confidence，应该放弃AutoNATRetryInterval
	// * recent inbound connections (implying continued connectivity) should decrease the retry when public
	// 当网络可达，应该减少尝试
	// * recent inbound connections when not public mean we should try more actively to see if we're public.
	// 最近连接的连接不可公网可达，我们应该积极的尝试
	fixedNow := time.Now()
	currentStatus := as.status.Load().(autoNATResult)

	nextProbe := fixedNow
	// Don't look for peers in the peer store more than once per second.
	if !as.lastProbeTry.IsZero() {
		backoff := as.lastProbeTry.Add(time.Second)
		if backoff.After(nextProbe) {
			nextProbe = backoff
		}
	}
	if !as.lastProbe.IsZero() {
		untilNext := as.config.refreshInterval
		if currentStatus.Reachability == network.ReachabilityUnknown { //当前位置，正常探索
			untilNext = as.config.retryInterval
		} else if as.confidence < 3 { // 信心不足，正常探索
			untilNext = as.config.retryInterval
		} else if currentStatus.Reachability == network.ReachabilityPublic && as.lastInbound.After(as.lastProbe) { // 网络可达，则积极减少探索
			untilNext *= 2
		} else if currentStatus.Reachability != network.ReachabilityPublic && as.lastInbound.After(as.lastProbe) { // 网络不可待，则积极探索
			untilNext /= 5
		}

		if as.lastProbe.Add(untilNext).After(nextProbe) {
			nextProbe = as.lastProbe.Add(untilNext)
		}
	}

	return nextProbe.Sub(fixedNow)
}

// Update the current status based on an observed result.
// 基于探测结果更新当前状态，
// 1. 从不可达，变成可达ReachabilityPublic，开启autonat服务confidence置位为0； 如果先前状态为可达或者未知，在confidence小于3的情况下，自增；
// 2. 从可达到不可达ReachabilityPrivate，如果先前为可达状态，在confidence大于0的情况下，减少confidence，如果confidence小于0，表示到其他节点也不同，关闭自动探测服务，confidence置位0；
// 如果先前状态为非可达状态，如果confidence小于3，自增confidence；
// 3. 如果为未知状态，，在confidence大于0的情况下，减少confidence，在先前状态为ReachabilityPrivate情况下，重启开启autonet服务；
func (as *AmbientAutoNAT) recordObservation(observation autoNATResult) {
	// 获取当前状态
	currentStatus := as.status.Load().(autoNATResult)
	if observation.Reachability == network.ReachabilityPublic { //网络可达
		log.Debugf("NAT status is public")
		changed := false
		if currentStatus.Reachability != network.ReachabilityPublic { //非可达状态
			// we are flipping our NATStatus, so confidence drops to 0
			as.confidence = 0
			if as.service != nil {
				//开启服务，设置协议流处理器
				as.service.Enable()
			}
			changed = true
		} else if as.confidence < 3 {
			as.confidence++ //autonat 可达信心指数+1
		}
		if observation.address != nil {
			if !changed && currentStatus.address != nil && !observation.address.Equal(currentStatus.address) {
				as.confidence--
			}
			if currentStatus.address == nil || !observation.address.Equal(currentStatus.address) { //新的候选者peer节点
				changed = true
			}
			//存储探测结果状态
			as.status.Store(observation)
		}
		if observation.address != nil && changed { //发出节点网络可达变更事件
			as.emitStatus()
		}
	} else if observation.Reachability == network.ReachabilityPrivate { //网络变成不可达，节点公网不可达， 但是节点仍有可能通过中继器可达
		log.Debugf("NAT status is private")
		if currentStatus.Reachability == network.ReachabilityPublic { //网络变成不可达
			if as.confidence > 0 { //可选中继节点信息指数--1
				as.confidence--
			} else {
				// we are flipping our NATStatus, so confidence drops to 0
				as.confidence = 0
				as.status.Store(observation)
				if as.service != nil { //不可达，则关闭autonat服务
					as.service.Disable()
				}
				as.emitStatus() //发出节点网络不可达变更事件
			}
		} else if as.confidence < 3 {
			as.confidence++ // 不可达
			as.status.Store(observation)
			if currentStatus.Reachability != network.ReachabilityPrivate {
				as.emitStatus()
			}
		}
	} else if as.confidence > 0 { //未知网络可达状态，信心指数自减
		// don't just flip to unknown, reduce confidence first
		as.confidence--
	} else { //未知网络可达状态
		log.Debugf("NAT status is unknown")
		as.status.Store(autoNATResult{network.ReachabilityUnknown, nil})
		if currentStatus.Reachability != network.ReachabilityUnknown {
			if as.service != nil {
				as.service.Enable() //开启服务
			}
			as.emitStatus()
		}
	}
}

// 尝试探测peer，发送DialBack探测peer，如果成功则网络可达，失败则不可达，默认为位置可达状态
func (as *AmbientAutoNAT) tryProbe(p peer.ID) bool {
	as.lastProbeTry = time.Now()
	if p.Validate() != nil {
		return false
	}

	if lastTime, ok := as.recentProbes[p]; ok { //退避时间间隔不够
		if time.Since(lastTime) < as.throttlePeerPeriod {
			return false
		}
	}
	// 清除最近探测的大于退避时间的peer
	as.cleanupRecentProbes()
	//获取当前peer信息
	info := as.host.Peerstore().PeerInfo(p)

	if !as.config.dialPolicy.skipPeer(info.Addrs) { //不能跳过探测peer，需要探测
		as.recentProbes[p] = time.Now()
		as.lastProbe = time.Now()
		go as.probe(&info) //启动探测，发送DialBack探测peer，如果成功则网络可达，失败则不可达，默认为位置可达状态
		return true
	}
	return false
}

// 发送DialBack探测peer，如果成功则网络可达（获取响应拨号的peer地址），失败则不可达，默认为位置可达状态
func (as *AmbientAutoNAT) probe(pi *peer.AddrInfo) {
	//创建AutoNATClient
	cli := NewAutoNATClient(as.host, as.config.addressFunc)
	//创建请求超时上下文
	ctx, cancel := context.WithTimeout(as.ctx, as.config.requestTimeout)
	defer cancel() //退出取消
	//回拨peer，获取响应拨号的peer地址
	a, err := cli.DialBack(ctx, pi.ID)

	var result autoNATResult
	switch {
	case err == nil: //网络可达
		log.Debugf("Dialback through %s successful; public address is %s", pi.ID.Pretty(), a.String())
		result.Reachability = network.ReachabilityPublic
		result.address = a
	case IsDialError(err): //网络不可达，私有
		log.Debugf("Dialback through %s failed", pi.ID.Pretty())
		result.Reachability = network.ReachabilityPrivate
	default: //未知可达状态
		result.Reachability = network.ReachabilityUnknown
	}

	select {
	case as.observations <- result: //将自动探测结果发送到observations
	case <-as.ctx.Done():
		return
	}
}

// 从peer的探测peer中选择一个探测节点
func (as *AmbientAutoNAT) getPeerToProbe() peer.ID {
	peers := as.host.Network().Peers() //获取当期节点已经连接的peers ？？？ 启动就有，还是autoRelay探测服务发现的
	if len(peers) == 0 {
		return ""
	}
	//候选者节点
	candidates := make([]peer.ID, 0, len(peers))
	//从peer中筛选支持autonat协议的候选者peer
	for _, p := range peers {
		info := as.host.Peerstore().PeerInfo(p)
		// Exclude peers which don't support the autonat protocol.
		// 剔出不支持autoNa协议的
		if proto, err := as.host.Peerstore().SupportsProtocols(p, AutoNATProto); len(proto) == 0 || err != nil {
			continue
		}

		// Exclude peers in backoff. 剔除已经探测过的节点
		if lastTime, ok := as.recentProbes[p]; ok {
			if time.Since(lastTime) < as.throttlePeerPeriod { //小于探测间隔
				continue
			}
		}

		if as.config.dialPolicy.skipPeer(info.Addrs) { // 跳过peer
			continue
		}
		candidates = append(candidates, p)
	}

	if len(candidates) == 0 {
		return ""
	}
	// 打乱peer
	shufflePeers(candidates)
	//选出第一个候选者
	return candidates[0]
}

func (as *AmbientAutoNAT) Close() error {
	as.ctxCancel()
	if as.service != nil {
		as.service.Disable()
	}
	<-as.backgroundRunning
	return nil
}

// 打乱peer
func shufflePeers(peers []peer.ID) {
	for i := range peers {
		j := rand.Intn(i + 1)
		peers[i], peers[j] = peers[j], peers[i]
	}
}

// Status returns the AutoNAT observed reachability status.
func (s *StaticAutoNAT) Status() network.Reachability {
	return s.reachability
}

// PublicAddr returns the publicly connectable Multiaddr of this node if one is known.
func (s *StaticAutoNAT) PublicAddr() (ma.Multiaddr, error) {
	if s.reachability != network.ReachabilityPublic {
		return nil, errors.New("NAT status is not public")
	}
	return nil, errors.New("no available address")
}

func (s *StaticAutoNAT) Close() error {
	if s.service != nil {
		s.service.Disable()
	}
	return nil
}
