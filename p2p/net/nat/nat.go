package nat

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/libp2p/go-nat"
)

// ErrNoMapping signals no mapping exists for an address
var ErrNoMapping = errors.New("mapping not established")

var log = logging.Logger("nat")

// MappingDuration is a default port mapping duration. 映射持续时间
// Port mappings are renewed every (MappingDuration / 3)
const MappingDuration = time.Second * 60

// CacheTime is the time a mapping will cache an external address for 外部地址缓存时间
const CacheTime = time.Second * 15

// DiscoverNAT looks for a NAT device in the network and
// returns an object that can manage port mappings.发现NAT设备，并返回管理NAT端口映射的对象
func DiscoverNAT(ctx context.Context) (*NAT, error) {
	natInstance, err := nat.DiscoverGateway(ctx)
	if err != nil {
		return nil, err
	}

	// Log the device addr.
	addr, err := natInstance.GetDeviceAddress()
	if err != nil {
		log.Debug("DiscoverGateway address error:", err)
	} else {
		log.Debug("DiscoverGateway address:", addr)
	}

	return newNAT(natInstance), nil
}

// NAT is an object that manages address port mappings in NAT管理端口映射的对象
// NATs (Network Address Translators). It is a long-running
// service that will periodically renew port mappings,
// and keep an up-to-date list of all the external addresses.
// 他是一个long-running服务，将会间歇性性地创新创建port映射，并保持所有地址的最新列表
type NAT struct {
	natmu sync.Mutex
	nat   nat.NAT

	refCount  sync.WaitGroup     //CountDownLock
	ctx       context.Context    //上下文
	ctxCancel context.CancelFunc //取消功能

	mappingmu sync.RWMutex          // guards mappings 读写锁保护
	closed    bool                  //是否关闭
	mappings  map[*mapping]struct{} //端口映射
}

// 创建NAT
func newNAT(realNAT nat.NAT) *NAT {
	ctx, cancel := context.WithCancel(context.Background())
	return &NAT{
		nat:       realNAT,
		mappings:  make(map[*mapping]struct{}),
		ctx:       ctx,
		ctxCancel: cancel,
	}
}

// Close shuts down all port mappings. NAT can no longer be used.
// 关闭所有端口映射，NAT不在使用
func (nat *NAT) Close() error {
	nat.mappingmu.Lock()
	nat.closed = true
	nat.mappingmu.Unlock()

	nat.ctxCancel()
	nat.refCount.Wait()
	return nil
}

// Mappings returns a slice of all NAT mappings 所有NAT的端口映射
func (nat *NAT) Mappings() []Mapping {
	nat.mappingmu.Lock()
	maps2 := make([]Mapping, 0, len(nat.mappings))
	for m := range nat.mappings {
		maps2 = append(maps2, m)
	}
	nat.mappingmu.Unlock()
	return maps2
}

// NewMapping attempts to construct a mapping on protocol and internal port NewMapping尝试构造一个基于协议、内部接口的映射；
// It will also periodically renew the mapping until the returned Mapping
// -- or its parent NAT -- is Closed. 同时也会间断性的重建接口映射或者返回父类的NAT
//
// May not succeed, and mappings may change over time; 由于端口映射随时可以改变，创建端口映射也许不会成功；
// NAT devices may not respect our port requests, and even lie. NAT设备也不会重试我们的请求，设置撒谎
// Clients should not store the mapped results, but rather always
// poll our object for the latest mappings. 客户端不应该存储映射接口，而是从对象中，拉取最新的映射
func (nat *NAT) NewMapping(protocol string, port int) (Mapping, error) {
	if nat == nil {
		return nil, fmt.Errorf("no nat available")
	}

	switch protocol {
	case "tcp", "udp": //支持tcp和udp
	default:
		return nil, fmt.Errorf("invalid protocol: %s", protocol)
	}
	//构建映射
	m := &mapping{
		intport: port,
		nat:     nat,
		proto:   protocol,
	}

	nat.mappingmu.Lock()
	if nat.closed { //NAT关闭
		nat.mappingmu.Unlock()
		return nil, errors.New("closed")
	}
	nat.mappings[m] = struct{}{}
	nat.refCount.Add(1) //计数器自增
	nat.mappingmu.Unlock()
	//刷新mapper
	go nat.refreshMappings(m)

	// do it once synchronously, so first mapping is done right away, and before exiting,
	// allowing users -- in the optimistic case -- to use results right after.
	// 同步建立映射，
	nat.establishMapping(m)
	return m, nil
}

// 移除NAT映射
func (nat *NAT) removeMapping(m *mapping) {
	nat.mappingmu.Lock()
	delete(nat.mappings, m)
	nat.mappingmu.Unlock()
	nat.natmu.Lock()
	nat.nat.DeletePortMapping(m.Protocol(), m.InternalPort())
	nat.natmu.Unlock()
}

// 刷新NAT映射
func (nat *NAT) refreshMappings(m *mapping) {
	defer nat.refCount.Done() //自减watiGroup
	// 延迟定时器
	t := time.NewTicker(MappingDuration / 3)
	defer t.Stop()

	for {
		select {
		case <-t.C: //建立连接
			nat.establishMapping(m)
		case <-nat.ctx.Done(): //nat 关闭
			m.Close()
			return
		}
	}
}

// 建议端口映射
func (nat *NAT) establishMapping(m *mapping) {
	oldport := m.ExternalPort()

	log.Debugf("Attempting port map: %s/%d", m.Protocol(), m.InternalPort())
	const comment = "libp2p"

	nat.natmu.Lock()
	//添加端口映射到NAT
	newport, err := nat.nat.AddPortMapping(m.Protocol(), m.InternalPort(), comment, MappingDuration)
	if err != nil {
		// Some hardware does not support mappings with timeout, so try that
		newport, err = nat.nat.AddPortMapping(m.Protocol(), m.InternalPort(), comment, 0)
	}
	nat.natmu.Unlock()

	if err != nil || newport == 0 {
		m.setExternalPort(0) // clear mapping
		// TODO: log.Event
		if err != nil {
			log.Warnf("failed to establish port mapping: %s", err)
		} else {
			log.Warnf("failed to establish port mapping: newport = 0")
		}
		// we do not close if the mapping failed,
		// because it may work again next time.
		return
	}
	//设置外部端口
	m.setExternalPort(newport)
	log.Debugf("NAT Mapping: %d --> %d (%s)", m.ExternalPort(), m.InternalPort(), m.Protocol())
	if oldport != 0 && newport != oldport {
		log.Debugf("failed to renew same port mapping: ch %d -> %d", oldport, newport)
	}
}
