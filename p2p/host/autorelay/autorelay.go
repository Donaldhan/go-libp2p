package autorelay

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	basic "github.com/libp2p/go-libp2p/p2p/host/basic"

	logging "github.com/ipfs/go-log/v2"
	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("autorelay")

type AutoRelay struct {
	refCount  sync.WaitGroup     //CountDownLock
	ctx       context.Context    //上下文
	ctxCancel context.CancelFunc //取消方法

	conf *config

	mx     sync.Mutex           //互斥锁
	status network.Reachability //是否可达

	relayFinder *relayFinder

	host   host.Host          //host
	addrsF basic.AddrsFactory //地址工厂
}

// 新建自动中继器
func NewAutoRelay(bhost *basic.BasicHost, opts ...Option) (*AutoRelay, error) {
	r := &AutoRelay{
		host:   bhost,
		addrsF: bhost.AddrsFactory,
		status: network.ReachabilityUnknown, //不可达
	}
	conf := defaultConfig
	for _, opt := range opts {
		if err := opt(&conf); err != nil {
			return nil, err
		}
	}
	r.ctx, r.ctxCancel = context.WithCancel(context.Background())
	r.conf = &conf
	//新建中继发现器，从配置的peer源
	r.relayFinder = newRelayFinder(bhost, conf.peerSource, &conf)
	bhost.AddrsFactory = r.hostAddrs

	r.refCount.Add(1) //CountDownLock 自增
	go func() {
		defer r.refCount.Done() //background执行完，CountDown
		r.background()
	}()
	return r, nil
}

// 探测节点可达性NAT，PMP， IGD，如果网络非ReachabilityPublic，则启动中继探测；如果可达，则停止中继探测
func (r *AutoRelay) background() {
	//订阅当本地节点可达状态变更时，emit event
	subReachability, err := r.host.EventBus().Subscribe(new(event.EvtLocalReachabilityChanged))
	if err != nil {
		log.Debug("failed to subscribe to the EvtLocalReachabilityChanged")
		return
	}
	defer subReachability.Close() //退出关闭订阅

	for {
		select {
		case <-r.ctx.Done():
			return
		case ev, ok := <-subReachability.Out():
			if !ok {
				return
			}
			// TODO: push changed addresses
			evt := ev.(event.EvtLocalReachabilityChanged)
			switch evt.Reachability {
			case network.ReachabilityPrivate, network.ReachabilityUnknown: //不可达
				if err := r.relayFinder.Start(); err != nil { //启动发现中继器
					log.Errorw("failed to start relay finder", "error", err)
				}
			case network.ReachabilityPublic: //网络可达，则停止发现中继器
				r.relayFinder.Stop()
			}
			r.mx.Lock()
			r.status = evt.Reachability
			r.mx.Unlock()
		}
	}
}

func (r *AutoRelay) hostAddrs(addrs []ma.Multiaddr) []ma.Multiaddr {
	return r.relayAddrs(r.addrsF(addrs))
}

func (r *AutoRelay) relayAddrs(addrs []ma.Multiaddr) []ma.Multiaddr {
	r.mx.Lock()
	defer r.mx.Unlock()

	if r.status != network.ReachabilityPrivate {
		return addrs
	}
	return r.relayFinder.relayAddrs(addrs)
}

func (r *AutoRelay) Close() error {
	r.ctxCancel()
	err := r.relayFinder.Stop()
	r.refCount.Wait()
	return err
}
