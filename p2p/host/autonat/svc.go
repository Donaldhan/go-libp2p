package autonat

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	pb "github.com/libp2p/go-libp2p/p2p/host/autonat/pb"

	"github.com/libp2p/go-msgio/protoio"
	ma "github.com/multiformats/go-multiaddr"
)

var streamTimeout = 60 * time.Second

const (
	ServiceName = "libp2p.autonat" //服务名

	maxMsgSize = 4096
)

// AutoNATService provides NAT autodetection services to other peers
// AutoNATService 提供NAT到其他peer的自动探测服务
type autoNATService struct {
	instanceLock      sync.Mutex         // 实例互斥锁
	instance          context.CancelFunc // 取消方法
	backgroundRunning chan struct{}      // closed when background exits hook

	config *config //

	// rate limiter
	mx         sync.Mutex      //
	reqs       map[peer.ID]int // 每个peer节点的请求
	globalReqs int             // 全局请求数
}

// NewAutoNATService creates a new AutoNATService instance attached to a host
// 创建NewAutoNATService
func newAutoNATService(c *config) (*autoNATService, error) {
	if c.dialer == nil {
		return nil, errors.New("cannot create NAT service without a network")
	}
	return &autoNATService{
		config: c,
		reqs:   make(map[peer.ID]int),
	}, nil
}

// 处理流
func (as *autoNATService) handleStream(s network.Stream) {
	if err := s.Scope().SetService(ServiceName); err != nil { //设置协议服务名失败
		log.Debugf("error attaching stream to autonat service: %s", err)
		s.Reset()
		return
	}
	//分配缓存空间
	if err := s.Scope().ReserveMemory(maxMsgSize, network.ReservationPriorityAlways); err != nil {
		log.Debugf("error reserving memory for autonat stream: %s", err)
		s.Reset()
		return
	}
	defer s.Scope().ReleaseMemory(maxMsgSize) //退出释放空间

	s.SetDeadline(time.Now().Add(streamTimeout)) //设置流超时时间
	defer s.Close()

	pid := s.Conn().RemotePeer() //获取流的远端peer
	log.Debugf("New stream from %s", pid.Pretty())
	//分割符读写器
	r := protoio.NewDelimitedReader(s, maxMsgSize)
	w := protoio.NewDelimitedWriter(s)

	var req pb.Message
	var res pb.Message

	err := r.ReadMsg(&req)
	if err != nil { //读取错误
		log.Debugf("Error reading message from %s: %s", pid.Pretty(), err.Error())
		s.Reset()
		return
	}

	t := req.GetType()
	if t != pb.Message_DIAL { //非拨号消息，错误
		log.Debugf("Unexpected message from %s: %s (%d)", pid.Pretty(), t.String(), t)
		s.Reset()
		return
	}
	//处理拨号请求(远端peerId，远端多播地址，peer信息)
	dr := as.handleDial(pid, s.Conn().RemoteMultiaddr(), req.GetDial().GetPeer())
	res.Type = pb.Message_DIAL_RESPONSE.Enum() //拨号响应消息
	res.DialResponse = dr

	err = w.WriteMsg(&res) //dial back
	if err != nil {
		log.Debugf("Error writing response to %s: %s", pid.Pretty(), err.Error())
		s.Reset()
		return
	}
}

// 处理拨号请求(远端peerId，远端多播地址，peer信息)
// 1.
func (as *autoNATService) handleDial(p peer.ID, obsaddr ma.Multiaddr, mpi *pb.Message_PeerInfo) *pb.Message_DialResponse {
	if mpi == nil { //拨号底子为空，bad请求，peer信息缺失
		return newDialResponseError(pb.Message_E_BAD_REQUEST, "missing peer info")
	}

	mpid := mpi.GetId()
	if mpid != nil {
		mp, err := peer.IDFromBytes(mpid)
		if err != nil { // peerID 错误
			return newDialResponseError(pb.Message_E_BAD_REQUEST, "bad peer id")
		}

		if mp != p { //PeerID，不匹配
			return newDialResponseError(pb.Message_E_BAD_REQUEST, "peer id mismatch")
		}
	}
	//远端多播地址
	addrs := make([]ma.Multiaddr, 0, as.config.maxPeerAddresses)
	seen := make(map[string]struct{})

	// Don't even try to dial peers with blocked remote addresses. In order to dial a peer, we
	// need to know their public IP address, and it needs to be different from our public IP
	// address. 不要超时block的远端地址，为了拨号peer，我们需要知道他们的公网地址，同时不同与我们的公钥ip；
	if as.config.dialPolicy.skipDial(obsaddr) { //判断是否需要跳过peer
		// Note: versions < v0.20.0 return Message_E_DIAL_ERROR here, thus we can not rely on this error code.
		return newDialResponseError(pb.Message_E_DIAL_REFUSED, "refusing to dial peer with blocked observed address")
	}

	// Determine the peer's IP address. 获取peer的Ip
	//
	hostIP, _ := ma.SplitFirst(obsaddr)
	//检查是否为合法的ip4，或ip6地址
	switch hostIP.Protocol().Code {
	case ma.P_IP4, ma.P_IP6:
	default:
		// This shouldn't be possible as we should skip all addresses that don't include
		// public IP addresses. 内部错误
		return newDialResponseError(pb.Message_E_INTERNAL_ERROR, "expected an IP address")
	}

	// add observed addr to the list of addresses to dial 添加地址到拨号地址
	addrs = append(addrs, obsaddr)
	seen[obsaddr.String()] = struct{}{}

	for _, maddr := range mpi.GetAddrs() {
		addr, err := ma.NewMultiaddrBytes(maddr) //转换多播地址
		if err != nil {
			log.Debugf("Error parsing multiaddr: %s", err.Error())
			continue
		}

		// For security reasons, we _only_ dial the observed IP address. 安全原因，我们需要拨号观察到ip地址，替代那些我们能需要探测的ip地址
		// Replace other IP addresses with the observed one so we can still try the
		// requested ports/transports.
		if ip, rest := ma.SplitFirst(addr); !ip.Equal(hostIP) {
			// Make sure it's an IP address
			switch ip.Protocol().Code {
			case ma.P_IP4, ma.P_IP6: //检查ip4，ip6
			default:
				continue
			}
			addr = hostIP
			if rest != nil {
				//封装多播地址 /ip4/1.2.3.4 encapsulate /tcp/80 = /ip4/1.2.3.4/tcp/80
				addr = addr.Encapsulate(rest)
			}
		}

		// Make sure we're willing to dial the rest of the address (e.g., not a circuit
		// address). 确认是否需要跳过peer地址
		if as.config.dialPolicy.skipDial(addr) {
			continue
		}

		str := addr.String()
		_, ok := seen[str]
		if ok {
			continue
		}

		addrs = append(addrs, addr)
		seen[str] = struct{}{}
		//可达的多播地址足够
		if len(addrs) >= as.config.maxPeerAddresses {
			break
		}
	}

	if len(addrs) == 0 {
		// 没有拨号地址
		// Note: versions < v0.20.0 return Message_E_DIAL_ERROR here, thus we can not rely on this error code.
		return newDialResponseError(pb.Message_E_DIAL_REFUSED, "no dialable addresses")
	}

	return as.doDial(peer.AddrInfo{ID: p, Addrs: addrs})
}

// 处理实际拨号
// 1. 统计拨号请求计数器
// 2. 清除peer先前存储的地址，更新autonet服务的peer拨号网络最新地址
func (as *autoNATService) doDial(pi peer.AddrInfo) *pb.Message_DialResponse {
	// rate limit check
	as.mx.Lock()
	count := as.reqs[pi.ID] //peerId的请求数
	if count >= as.config.throttlePeerMax || (as.config.throttleGlobalMax > 0 &&
		as.globalReqs >= as.config.throttleGlobalMax) {
		as.mx.Unlock() //ddos攻击，请求超时
		return newDialResponseError(pb.Message_E_DIAL_REFUSED, "too many dials")
	}
	as.reqs[pi.ID] = count + 1 //统计计数器
	as.globalReqs++            //全局请求+1
	as.mx.Unlock()
	//超时拨号
	ctx, cancel := context.WithTimeout(context.Background(), as.config.dialTimeout)
	defer cancel()
	// 清除peer先前存储的地址
	as.config.dialer.Peerstore().ClearAddrs(pi.ID)
	// 更新peer新地址
	as.config.dialer.Peerstore().AddAddrs(pi.ID, pi.Addrs, peerstore.TempAddrTTL)

	defer func() {
		as.config.dialer.Peerstore().ClearAddrs(pi.ID) //清除peer先前存储的地
		as.config.dialer.Peerstore().RemovePeer(pi.ID) // 更新peer新地址
	}()
	//拨号远端地址
	conn, err := as.config.dialer.DialPeer(ctx, pi.ID)
	if err != nil {
		log.Debugf("error dialing %s: %s", pi.ID.Pretty(), err.Error())
		// wait for the context to timeout to avoid leaking timing information
		// this renders the service ineffective as a port scanner
		<-ctx.Done()
		return newDialResponseError(pb.Message_E_DIAL_ERROR, "dial failed")
	}
	//连接远端多播地址
	ra := conn.RemoteMultiaddr()
	as.config.dialer.ClosePeer(pi.ID) //关闭peer连接
	return newDialResponseOK(ra)      // 拨号响应成功
}

// Enable the autoNAT service if it is not running.
// 如果autoNATService没有运行，则开启服务，设置NAT协议流处理器，同时启动后台清理NAT服务请求计数器后台协程；
func (as *autoNATService) Enable() {
	as.instanceLock.Lock()
	defer as.instanceLock.Unlock() //退出解锁
	if as.instance != nil {
		return
	}
	//取消上下文
	ctx, cancel := context.WithCancel(context.Background())
	as.instance = cancel
	as.backgroundRunning = make(chan struct{})
	//AutoNat协议流处理器
	as.config.host.SetStreamHandler(AutoNATProto, as.handleStream)
	//开启清除NAT服务请求计数器后台协程
	go as.background(ctx)
}

// Disable the autoNAT service if it is running.
// 关闭服务
func (as *autoNATService) Disable() {
	as.instanceLock.Lock()
	defer as.instanceLock.Unlock()
	if as.instance != nil {
		//移除NAT协议流处理器
		as.config.host.RemoveStreamHandler(AutoNATProto)
		as.instance()
		as.instance = nil
		<-as.backgroundRunning
	}
}

// 后台goroutine，清除NAT服务请求计数器，重置定时器
func (as *autoNATService) background(ctx context.Context) {
	defer close(as.backgroundRunning)

	timer := time.NewTimer(as.config.throttleResetPeriod)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			as.mx.Lock()
			//清除NAT服务请求计数器
			as.reqs = make(map[peer.ID]int)
			as.globalReqs = 0
			as.mx.Unlock()
			jitter := rand.Float32() * float32(as.config.throttleResetJitter)
			//重置定时器
			timer.Reset(as.config.throttleResetPeriod + time.Duration(int64(jitter)))
		case <-ctx.Done():
			return
		}
	}
}
