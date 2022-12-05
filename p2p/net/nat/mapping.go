package nat

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Mapping represents a port mapping in a NAT. 表示NAT中一个端口映射
type Mapping interface {
	// NAT returns the NAT object this Mapping belongs to.
	NAT() *NAT

	// Protocol returns the protocol of this port mapping. This is either
	// "tcp" or "udp" as no other protocols are likely to be NAT-supported.
	// 端口映射的协议
	Protocol() string

	// InternalPort returns the internal device port. Mapping will continue to
	// try to map InternalPort() to an external facing port.
	// 内部设备的端口
	InternalPort() int

	// ExternalPort returns the external facing port. If the mapping is not
	// established, port will be 0 外部端口
	ExternalPort() int

	// ExternalAddr returns the external facing address. If the mapping is not
	// established, addr will be nil, and and ErrNoMapping will be returned. 外部地址
	ExternalAddr() (addr net.Addr, err error)

	// Close closes the port mapping 关闭端口映射
	Close() error
}

// keeps republishing
type mapping struct {
	sync.Mutex // guards all fields 字段保护互斥锁

	nat     *NAT   //关联NAT
	proto   string //
	intport int    //内部端口
	extport int    //外部端口

	cached    net.IP     // ip地址
	cacheTime time.Time  // 映射时间
	cacheLk   sync.Mutex //
}

/// NAT端口映射信息接口

func (m *mapping) NAT() *NAT {
	m.Lock()
	defer m.Unlock() //延迟Unlock
	return m.nat
}

func (m *mapping) Protocol() string {
	m.Lock()
	defer m.Unlock()
	return m.proto
}

func (m *mapping) InternalPort() int {
	m.Lock()
	defer m.Unlock()
	return m.intport
}

func (m *mapping) ExternalPort() int {
	m.Lock()
	defer m.Unlock()
	return m.extport
}

func (m *mapping) setExternalPort(p int) {
	m.Lock()
	defer m.Unlock()
	m.extport = p
}

func (m *mapping) ExternalAddr() (net.Addr, error) {
	m.cacheLk.Lock()
	defer m.cacheLk.Unlock()
	oport := m.ExternalPort() //外部端口
	if oport == 0 {
		// dont even try right now.
		return nil, ErrNoMapping
	}

	if time.Since(m.cacheTime) >= CacheTime { //
		m.nat.natmu.Lock()
		cval, err := m.nat.nat.GetExternalAddress()
		m.nat.natmu.Unlock()

		if err != nil {
			return nil, err
		}

		m.cached = cval
		m.cacheTime = time.Now()
	}
	switch m.Protocol() {
	case "tcp":
		return &net.TCPAddr{
			IP:   m.cached,
			Port: oport,
		}, nil
	case "udp":
		return &net.UDPAddr{
			IP:   m.cached,
			Port: oport,
		}, nil
	default:
		panic(fmt.Sprintf("invalid protocol %q", m.Protocol()))
	}
}

func (m *mapping) Close() error {
	m.nat.removeMapping(m)
	return nil
}
