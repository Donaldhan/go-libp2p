package libp2p

// This file contains all libp2p configuration options (except the defaults,
// those are in defaults.go).

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/pnet"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/transport"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	tptu "github.com/libp2p/go-libp2p/p2p/net/upgrader"
	relayv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-libp2p/p2p/protocol/holepunch"
	"github.com/libp2p/go-libp2p/p2p/transport/quicreuse"

	ma "github.com/multiformats/go-multiaddr"
	madns "github.com/multiformats/go-multiaddr-dns"
	"go.uber.org/fx"
)

// ListenAddrStrings configures libp2p to listen on the given (unparsed)
// addresses. 监听地址
func ListenAddrStrings(s ...string) Option {
	return func(cfg *Config) error {
		for _, addrstr := range s {
			a, err := ma.NewMultiaddr(addrstr)
			if err != nil {
				return err
			}
			cfg.ListenAddrs = append(cfg.ListenAddrs, a)
		}
		return nil
	}
}

// ListenAddrs configures libp2p to listen on the given addresses.
func ListenAddrs(addrs ...ma.Multiaddr) Option {
	return func(cfg *Config) error {
		cfg.ListenAddrs = append(cfg.ListenAddrs, addrs...)
		return nil
	}
}

// Security configures libp2p to use the given security transport (or transport
// constructor).
// 构建安全的transport
// Name is the protocol name.
//
// The transport can be a constructed security.Transport or a function taking
// any subset of this libp2p node's:
// * Public key
// * Private key
// * Peer ID
// * Host
// * Network
// * Peerstore
func Security(name string, constructor interface{}) Option {
	return func(cfg *Config) error {
		if cfg.Insecure {
			return fmt.Errorf("cannot use security transports with an insecure libp2p configuration")
		}
		cfg.SecurityTransports = append(cfg.SecurityTransports, config.Security{ID: protocol.ID(name), Constructor: constructor})
		return nil
	}
}

// NoSecurity is an option that completely disables all transport security. 非安全的transport
// It's incompatible with all other transport security protocols.
var NoSecurity Option = func(cfg *Config) error {
	if len(cfg.SecurityTransports) > 0 {
		return fmt.Errorf("cannot use security transports with an insecure libp2p configuration")
	}
	cfg.Insecure = true
	return nil
}

// Muxer configures libp2p to use the given stream multiplexer. 多路复用器
// name is the protocol name.
func Muxer(name string, muxer network.Multiplexer) Option {
	return func(cfg *Config) error {
		cfg.Muxers = append(cfg.Muxers, tptu.StreamMuxer{Muxer: muxer, ID: protocol.ID(name)})
		return nil
	}
}

// QUIC
func QUICReuse(constructor interface{}, opts ...quicreuse.Option) Option {
	return func(cfg *Config) error {
		tag := `group:"quicreuseopts"`
		typ := reflect.ValueOf(constructor).Type()
		numParams := typ.NumIn()
		isVariadic := typ.IsVariadic()

		if !isVariadic && len(opts) > 0 {
			return errors.New("QUICReuse constructor doesn't take any options")
		}

		var params []string
		if isVariadic && len(opts) > 0 {
			// If there are options, apply the tag.
			// Since options are variadic, they have to be the last argument of the constructor.
			params = make([]string, numParams)
			params[len(params)-1] = tag
		}

		cfg.QUICReuse = append(cfg.QUICReuse, fx.Provide(fx.Annotate(constructor, fx.ParamTags(params...))))
		for _, opt := range opts {
			cfg.QUICReuse = append(cfg.QUICReuse, fx.Supply(fx.Annotate(opt, fx.ResultTags(tag))))
		}
		return nil
	}
}

// Transport configures libp2p to use the given transport (or transport
// constructor).
// Transport配置
// The transport can be a constructed transport.Transport or a function taking
// any subset of this libp2p node's:
// * Transport Upgrader (*tptu.Upgrader)
// * Host
// * Stream muxer (muxer.Transport)
// * Security transport (security.Transport)
// * Private network protector (pnet.Protector)
// * Peer ID
// * Private Key
// * Public Key
// * Address filter (filter.Filter)
// * Peerstore
func Transport(constructor interface{}, opts ...interface{}) Option {
	return func(cfg *Config) error {
		// generate a random identifier, so that fx can associate the constructor with its options
		b := make([]byte, 8)
		rand.Read(b)
		id := binary.BigEndian.Uint64(b)

		tag := fmt.Sprintf(`group:"transportopt_%d"`, id)

		typ := reflect.ValueOf(constructor).Type()
		numParams := typ.NumIn()
		isVariadic := typ.IsVariadic()

		if !isVariadic && len(opts) > 0 {
			return errors.New("transport constructor doesn't take any options")
		}
		if isVariadic && numParams >= 1 {
			paramType := typ.In(numParams - 1).Elem()
			for _, opt := range opts {
				if typ := reflect.TypeOf(opt); !typ.AssignableTo(paramType) {
					return fmt.Errorf("transport option of type %s not assignable to %s", typ, paramType)
				}
			}
		}

		var params []string
		if isVariadic && len(opts) > 0 {
			// If there are transport options, apply the tag.
			// Since options are variadic, they have to be the last argument of the constructor.
			params = make([]string, numParams)
			params[len(params)-1] = tag
		}

		cfg.Transports = append(cfg.Transports, fx.Provide(
			fx.Annotate(
				constructor,
				fx.ParamTags(params...),
				fx.As(new(transport.Transport)),
				fx.ResultTags(`group:"transport"`),
			),
		))
		for _, opt := range opts {
			cfg.Transports = append(cfg.Transports, fx.Supply(
				fx.Annotate(
					opt,
					fx.ResultTags(tag),
				),
			))
		}
		return nil
	}
}

// Peerstore configures libp2p to use the given peerstore. peerStore配置
func Peerstore(ps peerstore.Peerstore) Option {
	return func(cfg *Config) error {
		if cfg.Peerstore != nil {
			return fmt.Errorf("cannot specify multiple peerstore options")
		}

		cfg.Peerstore = ps
		return nil
	}
}

// PrivateNetwork configures libp2p to use the given private network protector. 私有网络
func PrivateNetwork(psk pnet.PSK) Option {
	return func(cfg *Config) error {
		if cfg.PSK != nil {
			return fmt.Errorf("cannot specify multiple private network options")
		}

		cfg.PSK = psk
		return nil
	}
}

// BandwidthReporter configures libp2p to use the given bandwidth reporter.
func BandwidthReporter(rep metrics.Reporter) Option {
	return func(cfg *Config) error {
		if cfg.Reporter != nil {
			return fmt.Errorf("cannot specify multiple bandwidth reporter options")
		}

		cfg.Reporter = rep
		return nil
	}
}

// Identity configures libp2p to use the given private key to identify itself. 身份Key
func Identity(sk crypto.PrivKey) Option {
	return func(cfg *Config) error {
		if cfg.PeerKey != nil {
			return fmt.Errorf("cannot specify multiple identities")
		}

		cfg.PeerKey = sk
		return nil
	}
}

// ConnectionManager configures libp2p to use the given connection manager. 连接管理器你
//
// The current "standard" connection manager lives in github.com/libp2p/go-libp2p-connmgr. See
// https://pkg.go.dev/github.com/libp2p/go-libp2p-connmgr?utm_source=godoc#NewConnManager.
func ConnectionManager(connman connmgr.ConnManager) Option {
	return func(cfg *Config) error {
		if cfg.ConnManager != nil {
			return fmt.Errorf("cannot specify multiple connection managers")
		}
		cfg.ConnManager = connman
		return nil
	}
}

// AddrsFactory configures libp2p to use the given address factory.
func AddrsFactory(factory config.AddrsFactory) Option {
	return func(cfg *Config) error {
		if cfg.AddrsFactory != nil {
			return fmt.Errorf("cannot specify multiple address factories")
		}
		cfg.AddrsFactory = factory
		return nil
	}
}

// EnableRelay configures libp2p to enable the relay transport. 开启中继器
// This option only configures libp2p to accept inbound connections from relays
// and make outbound connections_through_ relays when requested by the remote peer.
// This option supports both circuit v1 and v2 connections.
// (default: enabled)
func EnableRelay() Option {
	return func(cfg *Config) error {
		cfg.RelayCustom = true
		cfg.Relay = true
		return nil
	}
}

// DisableRelay configures libp2p to disable the relay transport.
func DisableRelay() Option {
	return func(cfg *Config) error {
		cfg.RelayCustom = true
		cfg.Relay = false
		return nil
	}
}

// EnableRelayService configures libp2p to run a circuit v2 relay, 开启中继服务
// if we detect that we're publicly reachable.
func EnableRelayService(opts ...relayv2.Option) Option {
	return func(cfg *Config) error {
		cfg.EnableRelayService = true
		cfg.RelayServiceOpts = opts
		return nil
	}
}

// EnableAutoRelay configures libp2p to enable the AutoRelay subsystem.
// 开启中继延迟服务， 配置示例
// Dependencies:
//   - Relay (enabled by default)
//   - Either:
//     1. A list of static relays 静态中继器地址
//     2. A PeerSource function that provides a chan of relays. See `autorelay.WithPeerSource` //提供中继服务的地址
//
// This subsystem performs automatic address rewriting to advertise relay addresses when it
// detects that the node is publicly unreachable (e.g. behind a NAT).
func EnableAutoRelay(opts ...autorelay.Option) Option {
	return func(cfg *Config) error {
		cfg.EnableAutoRelay = true
		cfg.AutoRelayOpts = opts
		return nil
	}
}

// ForceReachabilityPublic overrides automatic reachability detection in the AutoNAT subsystem,
// forcing the local node to believe it is reachable externally. 强制公开可达
func ForceReachabilityPublic() Option {
	return func(cfg *Config) error {
		public := network.Reachability(network.ReachabilityPublic)
		cfg.AutoNATConfig.ForceReachability = &public
		return nil
	}
}

// ForceReachabilityPrivate overrides automatic reachability detection in the AutoNAT subsystem,
// forceing the local node to believe it is behind a NAT and not reachable externally. 强制私有可达
func ForceReachabilityPrivate() Option {
	return func(cfg *Config) error {
		private := network.Reachability(network.ReachabilityPrivate)
		cfg.AutoNATConfig.ForceReachability = &private
		return nil
	}
}

// EnableNATService configures libp2p to provide a service to peers for determining
// their reachability status. When enabled, the host will attempt to dial back
// to peers, and then tell them if it was successful in making such connections.
// 开NAT服务配置，以便提供决定peer是否可达状态。 当开启时，host将会尝试dial back to peer；
func EnableNATService() Option {
	return func(cfg *Config) error {
		cfg.AutoNATConfig.EnableService = true
		return nil
	}
}

// AutoNATServiceRateLimit changes the default rate limiting configured in helping
// other peers determine their reachability status. When set, the host will limit
// the number of requests it responds to in each 60 second period to the set
// numbers. A value of '0' disables throttling. autonat限制配置
func AutoNATServiceRateLimit(global, perPeer int, interval time.Duration) Option {
	return func(cfg *Config) error {
		cfg.AutoNATConfig.ThrottleGlobalLimit = global
		cfg.AutoNATConfig.ThrottlePeerLimit = perPeer
		cfg.AutoNATConfig.ThrottleInterval = interval
		return nil
	}
}

// ConnectionGater configures libp2p to use the given ConnectionGater 连接网关
// to actively reject inbound/outbound connections based on the lifecycle stage
// of the connection.
//
// For more information, refer to go-libp2p/core.ConnectionGater.
func ConnectionGater(cg connmgr.ConnectionGater) Option {
	return func(cfg *Config) error {
		if cfg.ConnectionGater != nil {
			return errors.New("cannot configure multiple connection gaters, or cannot configure both Filters and ConnectionGater")
		}
		cfg.ConnectionGater = cg
		return nil
	}
}

// ResourceManager configures libp2p to use the given ResourceManager. 资源管理器
// When using the p2p/host/resource-manager implementation of the ResourceManager interface,
// it is recommended to set limits for libp2p protocol by calling SetDefaultServiceLimits.
func ResourceManager(rcmgr network.ResourceManager) Option {
	return func(cfg *Config) error {
		if cfg.ResourceManager != nil {
			return errors.New("cannot configure multiple resource managers")
		}
		cfg.ResourceManager = rcmgr
		return nil
	}
}

// NATPortMap configures libp2p to use the default NATManager. The default
// NATManager will attempt to open a port in your network's firewall using UPnP.
// https://zhuanlan.zhihu.com/p/40407669
// 使用NATPortMap配置libp2p的默认NATManager管理器，默认情况下，NATManager将会尝试在你的
// 网络防火墙，基于UPnP打开一个端口
func NATPortMap() Option {
	return NATManager(bhost.NewNATManager)
}

// NATManager will configure libp2p to use the requested NATManager. This NAT管理器
// function should be passed a NATManager *constructor* that takes a libp2p Network.
func NATManager(nm config.NATManagerC) Option {
	return func(cfg *Config) error {
		if cfg.NATManager != nil {
			return fmt.Errorf("cannot specify multiple NATManagers")
		}
		cfg.NATManager = nm
		return nil
	}
}

// Ping will configure libp2p to support the ping service; enable by default. 开启ping服务
func Ping(enable bool) Option {
	return func(cfg *Config) error {
		cfg.DisablePing = !enable
		return nil
	}
}

// Routing will configure libp2p to use routing. 路由配置
func Routing(rt config.RoutingC) Option {
	return func(cfg *Config) error {
		if cfg.Routing != nil {
			return fmt.Errorf("cannot specify multiple routing options")
		}
		cfg.Routing = rt
		return nil
	}
}

// NoListenAddrs will configure libp2p to not listen by default.
//
// This will both clear any configured listen addrs and prevent libp2p from
// applying the default listen address option. It also disables relay, unless the
// user explicitly specifies with an option, as the transport creates an implicit
// listen address that would make the node dialable through any relay it was connected to.
var NoListenAddrs = func(cfg *Config) error {
	cfg.ListenAddrs = []ma.Multiaddr{}
	if !cfg.RelayCustom {
		cfg.RelayCustom = true
		cfg.Relay = false
	}
	return nil
}

// NoTransports will configure libp2p to not enable any transports.
//
// This will both clear any configured transports (specified in prior libp2p
// options) and prevent libp2p from applying the default transports.  无transport
var NoTransports = func(cfg *Config) error {
	cfg.Transports = []fx.Option{}
	return nil
}

// ProtocolVersion sets the protocolVersion string required by the 配置协议版本
// libp2p Identify protocol.
func ProtocolVersion(s string) Option {
	return func(cfg *Config) error {
		cfg.ProtocolVersion = s
		return nil
	}
}

// UserAgent sets the libp2p user-agent sent along with the identify protocol 使用代理
func UserAgent(userAgent string) Option {
	return func(cfg *Config) error {
		cfg.UserAgent = userAgent
		return nil
	}
}

// MultiaddrResolver sets the libp2p dns resolver 多播解决器
func MultiaddrResolver(rslv *madns.Resolver) Option {
	return func(cfg *Config) error {
		cfg.MultiaddrResolver = rslv
		return nil
	}
}

// [家庭网络中的「NAT」到底是什么](https://zhuanlan.zhihu.com/p/410801432)
// [互联网网关设备协议IGD(Internet Gateway Device)](https://zh.wikipedia.org/wiki/%E4%BA%92%E8%81%94%E7%BD%91%E7%BD%91%E5%85%B3%E8%AE%BE%E5%A4%87%E5%8D%8F%E8%AE%AE)
// [NAT端口映射协议NAT-PMP](https://zh.wikipedia.org/wiki/NAT%E7%AB%AF%E5%8F%A3%E6%98%A0%E5%B0%84%E5%8D%8F%E8%AE%AE)
// [P2P网络编程-1-从分布式Hash表到LibP2P包的基本概念与使用](https://blog.csdn.net/weixin_43988498/article/details/118878439)
// [网络技术P2P技术原理浅析](https://keenjin.github.io/2021/04/p2p/)
// [P2P通信原理与实现](https://zhuanlan.zhihu.com/p/26796476)
// Experimental
// EnableHolePunching enables NAT traversal by enabling NATT'd peers to both initiate and respond to hole punching attempts
// to create direct/NAT-traversed connections with other peers. (default: disabled)
// 开启NAT打洞
// Dependencies:
//   - Relay (enabled by default)
//
// This subsystem performs two functions:
//
//  1. On receiving an inbound Relay connection, it attempts to create a direct connection with the remote peer
//     by initiating and co-ordinating a hole punch over the Relayed connection.
//  2. If a peer sees a request to co-ordinate a hole punch on an outbound Relay connection,
//     it will participate in the hole-punch to create a direct connection with the remote peer.
//
// If the hole punch is successful, all new streams will thereafter be created on the hole-punched connection.
// The Relayed connection will eventually be closed after a grace period.
//
// All existing indefinite long-lived streams on the Relayed connection will have to re-opened on the hole-punched connection by the user.
// Users can make use of the `Connected`/`Disconnected` notifications emitted by the Network for this purpose.
//
// It is not mandatory but nice to also enable the `AutoRelay` option (See `EnableAutoRelay`)
// so the peer can discover and connect to Relay servers  if it discovers that it is NATT'd and has private reachability via AutoNAT.
// This will then enable it to advertise Relay addresses which can be used to accept inbound Relay connections to then co-ordinate
// a hole punch.
//
// If `EnableAutoRelay` is configured and the user is confident that the peer has private reachability/is NATT'd,
// the `ForceReachabilityPrivate` option can be configured to short-circuit reachability discovery via AutoNAT
// so the peer can immediately start connecting to Relay servers.
//
// If `EnableAutoRelay` is configured, the `StaticRelays` option can be used to configure a static set of Relay servers
// for `AutoRelay` to connect to so that it does not need to discover Relay servers via Routing.
func EnableHolePunching(opts ...holepunch.Option) Option {
	return func(cfg *Config) error {
		cfg.EnableHolePunching = true
		cfg.HolePunchingOptions = opts
		return nil
	}
}

func WithDialTimeout(t time.Duration) Option {
	return func(cfg *Config) error {
		if t <= 0 {
			return errors.New("dial timeout needs to be non-negative")
		}
		cfg.DialTimeout = t
		return nil
	}
}
