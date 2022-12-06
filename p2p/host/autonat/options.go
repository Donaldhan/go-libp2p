package autonat

import (
	"errors"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

// config holds configurable options for the autonat subsystem.
// autonet系统的配置
type config struct {
	host host.Host // host 本机服务

	addressFunc       AddrFunc             //本地host对的候选地址函数
	dialPolicy        dialPolicy           //拨号策略
	dialer            network.Network      //拨号者
	forceReachability bool                 //强制可达
	reachability      network.Reachability // 网络可达性

	// client 客户端配置
	bootDelay          time.Duration // boot延迟
	retryInterval      time.Duration // 重试间隔
	refreshInterval    time.Duration // 刷新间隔
	requestTimeout     time.Duration // 请求超时时间
	throttlePeerPeriod time.Duration // peer节流时间，退避间隔

	// server 服务端配置
	dialTimeout         time.Duration //拨号超时时间
	maxPeerAddresses    int           //最大peer地址
	throttleGlobalMax   int           //
	throttlePeerMax     int           //
	throttleResetPeriod time.Duration //
	throttleResetJitter time.Duration //
}

// 默认配置
var defaults = func(c *config) error {
	c.bootDelay = 15 * time.Second
	c.retryInterval = 90 * time.Second
	c.refreshInterval = 15 * time.Minute
	c.requestTimeout = 30 * time.Second
	c.throttlePeerPeriod = 90 * time.Second

	c.dialTimeout = 15 * time.Second
	c.maxPeerAddresses = 16
	c.throttleGlobalMax = 30
	c.throttlePeerMax = 3
	c.throttleResetPeriod = 1 * time.Minute
	c.throttleResetJitter = 15 * time.Second
	return nil
}

// EnableService specifies that AutoNAT should be allowed to run a NAT service to help
// other peers determine their own NAT status. The provided Network should not be the
// default network/dialer of the host passed to `New`, as the NAT system will need to
// make parallel connections, and as such will modify both the associated peerstore
// and terminate connections of this dialer. The dialer provided
// should be compatible (TCP/UDP) however with the transports of the libp2p network.
// AutoNAT的EnableService服务，应该允许运行一个NAT服务，帮助其他peer节点确认网络状态。
// 提供的网络，不能为默认的host的默认network/dialer，由于NAT系统需要并行连接，这样，将会
// 修改peer及关联peer的peerstore。dialer应该提供与libp2p协议的兼容(TCP/UDP) transports
func EnableService(dialer network.Network) Option {
	return func(c *config) error {
		if dialer == c.host.Network() || dialer.Peerstore() == c.host.Peerstore() { // 拨号网络不能为自己
			return errors.New("dialer should not be that of the host")
		}
		c.dialer = dialer //拨号节点
		return nil
	}
}

// WithReachability overrides autonat to simply report an over-ridden reachability
// status. 可达， 重写autonat节点状态
func WithReachability(reachability network.Reachability) Option {
	return func(c *config) error {
		c.forceReachability = true
		c.reachability = reachability
		return nil
	}
}

// UsingAddresses allows overriding which Addresses the AutoNAT client believes
// are "its own". Useful for testing, or for more exotic port-forwarding
// scenarios where the host may be listening on different ports than it wants
// to externally advertise or verify connectability on. 可以地址方法
func UsingAddresses(addrFunc AddrFunc) Option {
	return func(c *config) error {
		if addrFunc == nil {
			return errors.New("invalid address function supplied")
		}
		c.addressFunc = addrFunc
		return nil
	}
}

// WithSchedule configures how agressively probes will be made to verify the
// address of the host. retryInterval indicates how often probes should be made
// when the host lacks confident about its address, while refresh interval
// is the schedule of periodic probes when the host believes it knows its
// steady-state reachability. 配置探测间隔
func WithSchedule(retryInterval, refreshInterval time.Duration) Option {
	return func(c *config) error {
		c.retryInterval = retryInterval
		c.refreshInterval = refreshInterval
		return nil
	}
}

// WithoutStartupDelay removes the initial delay the NAT subsystem typically
// uses as a buffer for ensuring that connectivity and guesses as to the hosts
// local interfaces have settled down during startup. 启动延迟
func WithoutStartupDelay() Option {
	return func(c *config) error {
		c.bootDelay = 1
		return nil
	}
}

// WithoutThrottling indicates that this autonat service should not place
// restrictions on how many peers it is willing to help when acting as
// a server. 退避探测时间
func WithoutThrottling() Option {
	return func(c *config) error {
		c.throttleGlobalMax = 0
		return nil
	}
}

// WithThrottling specifies how many peers (`amount`) it is willing to help
// ever `interval` amount of time when acting as a server.
func WithThrottling(amount int, interval time.Duration) Option {
	return func(c *config) error {
		c.throttleGlobalMax = amount
		c.throttleResetPeriod = interval
		c.throttleResetJitter = interval / 4
		return nil
	}
}

// WithPeerThrottling specifies a limit for the maximum number of IP checks
// this node will provide to an individual peer in each `interval`.
func WithPeerThrottling(amount int) Option {
	return func(c *config) error {
		c.throttlePeerMax = amount
		return nil
	}
}
