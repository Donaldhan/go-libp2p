package config

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/pnet"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/core/sec"
	"github.com/libp2p/go-libp2p/core/sec/insecure"
	"github.com/libp2p/go-libp2p/core/transport"
	"github.com/libp2p/go-libp2p/p2p/host/autonat"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	blankhost "github.com/libp2p/go-libp2p/p2p/host/blank"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	routed "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	tptu "github.com/libp2p/go-libp2p/p2p/net/upgrader"
	circuitv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	relayv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-libp2p/p2p/protocol/holepunch"
	"github.com/libp2p/go-libp2p/p2p/transport/quicreuse"

	ma "github.com/multiformats/go-multiaddr"
	madns "github.com/multiformats/go-multiaddr-dns"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

// AddrsFactory is a function that takes a set of multiaddrs we're listening on and
// returns the set of multiaddrs we should advertise to the network.
type AddrsFactory = bhost.AddrsFactory

// NATManagerC is a NATManager constructor. Nat管理器
type NATManagerC func(network.Network) bhost.NATManager

// host路由
type RoutingC func(host.Host) (routing.PeerRouting, error)

// AutoNATConfig defines the AutoNAT behavior for the libp2p host.
type AutoNATConfig struct {
	ForceReachability   *network.Reachability
	EnableService       bool          //autonat 启用装填
	ThrottleGlobalLimit int           //
	ThrottlePeerLimit   int           //peer限制阈值
	ThrottleInterval    time.Duration //阈值刷新间隔？？
}

type Security struct {
	ID          protocol.ID
	Constructor interface{}
}

// Config describes a set of settings for a libp2p node
// p2p节点的配置秒速
// This is *not* a stable interface. Use the options defined in the root
// package.
type Config struct {
	// UserAgent is the identifier this node will send to other peers when
	// identifying itself, e.g. via the identify protocol.
	// identify协议下的用户代理
	// Set it via the UserAgent option function.
	UserAgent string

	// ProtocolVersion is the protocol version that identifies the family
	// of protocols used by the peer in the Identify protocol. It is set
	// using the [ProtocolVersion] option. 协议版本
	ProtocolVersion string

	PeerKey crypto.PrivKey

	QUICReuse          []fx.Option
	Transports         []fx.Option        //传输端口
	Muxers             []tptu.StreamMuxer //流多路复用器
	SecurityTransports []Security
	Insecure           bool
	PSK                pnet.PSK

	DialTimeout time.Duration //拨号超时时间

	RelayCustom bool
	Relay       bool // should the relay transport be used

	EnableRelayService bool // should we run a circuitv2 relay (if publicly reachable)
	RelayServiceOpts   []relayv2.Option

	ListenAddrs     []ma.Multiaddr
	AddrsFactory    bhost.AddrsFactory //地址工厂
	ConnectionGater connmgr.ConnectionGater

	ConnManager     connmgr.ConnManager     //连接管理器
	ResourceManager network.ResourceManager //资源管理器

	NATManager NATManagerC         //Nat管理
	Peerstore  peerstore.Peerstore //Peer存储
	Reporter   metrics.Reporter

	MultiaddrResolver *madns.Resolver

	DisablePing bool //关闭Ping

	Routing RoutingC //peer路由

	EnableAutoRelay bool
	AutoRelayOpts   []autorelay.Option //autorelay 中继器配置
	AutoNATConfig                      //auto nat config 配置

	EnableHolePunching  bool
	HolePunchingOptions []holepunch.Option
}

func (cfg *Config) makeSwarm() (*swarm.Swarm, error) {
	if cfg.Peerstore == nil { //检查peer store
		return nil, fmt.Errorf("no peerstore specified")
	}

	// Check this early. Prevents us from even *starting* without verifying this.
	if pnet.ForcePrivateNetwork && len(cfg.PSK) == 0 {
		log.Error("tried to create a libp2p node with no Private" +
			" Network Protector but usage of Private Networks" +
			" is forced by the enviroment")
		// Note: This is *also* checked the upgrader itself so it'll be
		// enforced even *if* you don't use the libp2p constructor.
		return nil, pnet.ErrNotInPrivateNetwork
	}

	if cfg.PeerKey == nil { //检查peer key
		return nil, fmt.Errorf("no peer key specified")
	}

	// Obtain Peer ID from public key 从取peer的 public key 获取peerID
	pid, err := peer.IDFromPublicKey(cfg.PeerKey.GetPublic())
	if err != nil {
		return nil, err
	}
	//添加公私钥到Peerstore
	if err := cfg.Peerstore.AddPrivKey(pid, cfg.PeerKey); err != nil {
		return nil, err
	}
	if err := cfg.Peerstore.AddPubKey(pid, cfg.PeerKey.GetPublic()); err != nil {
		return nil, err
	}

	opts := make([]swarm.Option, 0, 3)
	if cfg.Reporter != nil {
		opts = append(opts, swarm.WithMetrics(cfg.Reporter))
	}
	if cfg.ConnectionGater != nil {
		opts = append(opts, swarm.WithConnectionGater(cfg.ConnectionGater))
	}
	if cfg.DialTimeout != 0 {
		//Dial超时
		opts = append(opts, swarm.WithDialTimeout(cfg.DialTimeout))
	}
	if cfg.ResourceManager != nil { //资源管理器
		opts = append(opts, swarm.WithResourceManager(cfg.ResourceManager))
	}
	if cfg.MultiaddrResolver != nil { //多播地址解决器
		opts = append(opts, swarm.WithMultiaddrResolver(cfg.MultiaddrResolver))
	}
	// TODO: Make the swarm implementation configurable. 创建Swarm
	//https://github.com/ethersphere/swarm
	return swarm.NewSwarm(pid, cfg.Peerstore, opts...)
}

// 添加host
func (cfg *Config) addTransports(h host.Host) error {
	swrm, ok := h.Network().(transport.TransportNetwork)
	if !ok {
		// Should probably skip this if no transports.
		return fmt.Errorf("swarm does not support transports")
	}

	fxopts := []fx.Option{
		fx.WithLogger(func() fxevent.Logger { return getFXLogger() }),
		fx.Provide(fx.Annotate(tptu.New, fx.ParamTags(`name:"security"`))),
		fx.Supply(cfg.Muxers),
		fx.Supply(h.ID()),
		fx.Provide(func() host.Host { return h }), //Host
		fx.Provide(func() crypto.PrivKey { return h.Peerstore().PrivKey(h.ID()) }), //私钥
		fx.Provide(func() connmgr.ConnectionGater { return cfg.ConnectionGater }),
		fx.Provide(func() pnet.PSK { return cfg.PSK }),
		fx.Provide(func() network.ResourceManager { return cfg.ResourceManager }), //资源管理器
		fx.Provide(func() *madns.Resolver { return cfg.MultiaddrResolver }),
	}
	fxopts = append(fxopts, cfg.Transports...)
	if cfg.Insecure { //安全
		fxopts = append(fxopts,
			fx.Provide(
				fx.Annotate(
					func(id peer.ID, priv crypto.PrivKey) []sec.SecureTransport {
						return []sec.SecureTransport{insecure.NewWithIdentity(insecure.ID, id, priv)}
					},
					fx.ResultTags(`name:"security"`),
				),
			),
		)
	} else {
		// fx groups are unordered, but we need to preserve the order of the security transports
		// First of all, we construct the security transports that are needed,
		// and save them to a group call security_unordered.
		for _, s := range cfg.SecurityTransports {
			fxName := fmt.Sprintf(`name:"security_%s"`, s.ID)
			fxopts = append(fxopts, fx.Supply(fx.Annotate(s.ID, fx.ResultTags(fxName))))
			fxopts = append(fxopts,
				fx.Provide(fx.Annotate(
					s.Constructor,
					fx.ParamTags(fxName),
					fx.As(new(sec.SecureTransport)),
					fx.ResultTags(`group:"security_unordered"`),
				)),
			)
		}
		// Then we consume the group security_unordered, and order them by the user's preference.
		fxopts = append(fxopts, fx.Provide(
			fx.Annotate(
				func(secs []sec.SecureTransport) ([]sec.SecureTransport, error) {
					if len(secs) != len(cfg.SecurityTransports) {
						return nil, errors.New("inconsistent length for security transports")
					}
					t := make([]sec.SecureTransport, 0, len(secs))
					for _, s := range cfg.SecurityTransports {
						for _, st := range secs {
							if s.ID != st.ID() {
								continue
							}
							t = append(t, st)
						}
					}
					return t, nil
				},
				fx.ParamTags(`group:"security_unordered"`),
				fx.ResultTags(`name:"security"`),
			)))
	}

	fxopts = append(fxopts, fx.Provide(PrivKeyToStatelessResetKey))
	if cfg.QUICReuse != nil {
		fxopts = append(fxopts, cfg.QUICReuse...)
	} else {
		fxopts = append(fxopts, fx.Provide(quicreuse.NewConnManager)) // TODO: close the ConnManager when shutting down the node
	}

	fxopts = append(fxopts, fx.Invoke(
		fx.Annotate(
			func(tpts []transport.Transport) error {
				for _, t := range tpts {
					if err := swrm.AddTransport(t); err != nil {
						return err
					}
				}
				return nil
			},
			fx.ParamTags(`group:"transport"`),
		)),
	)
	if cfg.Relay {
		fxopts = append(fxopts, fx.Invoke(circuitv2.AddTransport))
	}
	app := fx.New(fxopts...)
	if err := app.Err(); err != nil {
		h.Close()
		return err
	}
	return nil
}

// NewNode constructs a new libp2p Host from the Config.
//
// This function consumes the config. Do not reuse it (really!).
// 创建node，同时开启中继探测服务
func (cfg *Config) NewNode() (host.Host, error) {
	swrm, err := cfg.makeSwarm()
	if err != nil {
		return nil, err
	}
	//创建host
	h, err := bhost.NewHost(swrm, &bhost.HostOpts{
		ConnManager:         cfg.ConnManager, //连接管理器
		AddrsFactory:        cfg.AddrsFactory,
		NATManager:          cfg.NATManager,
		EnablePing:          !cfg.DisablePing,
		UserAgent:           cfg.UserAgent,
		ProtocolVersion:     cfg.ProtocolVersion,
		EnableHolePunching:  cfg.EnableHolePunching,
		HolePunchingOptions: cfg.HolePunchingOptions,
		EnableRelayService:  cfg.EnableRelayService,
		RelayServiceOpts:    cfg.RelayServiceOpts,
	})
	if err != nil {
		swrm.Close()
		return nil, err
	}

	if cfg.Relay { //中继器
		// If we've enabled the relay, we should filter out relay
		// addresses by default.
		//
		// TODO: We shouldn't be doing this here.
		oldFactory := h.AddrsFactory
		h.AddrsFactory = func(addrs []ma.Multiaddr) []ma.Multiaddr {
			return oldFactory(autorelay.Filter(addrs))
		}
	}
	//添加host到Transports
	if err := cfg.addTransports(h); err != nil {
		h.Close()
		return nil, err
	}

	// TODO: This method succeeds if listening on one address succeeds. We
	// should probably fail if listening on *any* addr fails. 开启监听
	if err := h.Network().Listen(cfg.ListenAddrs...); err != nil {
		h.Close()
		return nil, err
	}

	// Configure routing and autorelay 配置路由，自动安驰
	var router routing.PeerRouting
	if cfg.Routing != nil { //host路由
		router, err = cfg.Routing(h)
		if err != nil {
			h.Close()
			return nil, err
		}
	}

	// Note: h.AddrsFactory may be changed by relayFinder, but non-relay version is
	// used by AutoNAT below. host的地址工厂可以被relayFinder修改，但是非延迟版本使用如下AutoNAT
	var ar *autorelay.AutoRelay
	addrF := h.AddrsFactory
	if cfg.EnableAutoRelay { //开启自动中继
		if !cfg.Relay {
			h.Close()
			return nil, fmt.Errorf("cannot enable autorelay; relay is not enabled")
		}
		//创建自动中继服务，发现中继器
		ar, err = autorelay.NewAutoRelay(h, cfg.AutoRelayOpts...)
		if err != nil {
			return nil, err
		}
	}

	autonatOpts := []autonat.Option{
		autonat.UsingAddresses(func() []ma.Multiaddr {
			return addrF(h.AllAddrs())
		}),
	}
	if cfg.AutoNATConfig.ThrottleInterval != 0 {
		autonatOpts = append(autonatOpts,
			autonat.WithThrottling(cfg.AutoNATConfig.ThrottleGlobalLimit, cfg.AutoNATConfig.ThrottleInterval),
			autonat.WithPeerThrottling(cfg.AutoNATConfig.ThrottlePeerLimit))
	}
	if cfg.AutoNATConfig.EnableService {
		//生成公私钥
		autonatPrivKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return nil, err
		}
		ps, err := pstoremem.NewPeerstore()
		if err != nil {
			return nil, err
		}

		// Pull out the pieces of the config that we _actually_ care about.
		// Specifically, don't setup things like autorelay, listeners,
		// identify, etc.
		autoNatCfg := Config{
			Transports:         cfg.Transports,
			Muxers:             cfg.Muxers,
			SecurityTransports: cfg.SecurityTransports,
			Insecure:           cfg.Insecure,
			PSK:                cfg.PSK,
			ConnectionGater:    cfg.ConnectionGater,
			Reporter:           cfg.Reporter,
			PeerKey:            autonatPrivKey,
			Peerstore:          ps,
		}

		dialer, err := autoNatCfg.makeSwarm()
		if err != nil {
			h.Close()
			return nil, err
		}
		dialerHost := blankhost.NewBlankHost(dialer)
		if err := autoNatCfg.addTransports(dialerHost); err != nil {
			dialerHost.Close()
			h.Close()
			return nil, err
		}
		// NOTE: We're dropping the blank host here but that's fine. It
		// doesn't really _do_ anything and doesn't even need to be
		// closed (as long as we close the underlying network).
		autonatOpts = append(autonatOpts, autonat.EnableService(dialerHost.Network()))
	}
	if cfg.AutoNATConfig.ForceReachability != nil {
		autonatOpts = append(autonatOpts, autonat.WithReachability(*cfg.AutoNATConfig.ForceReachability))
	}
	//创建autonat
	autonat, err := autonat.New(h, autonatOpts...)
	if err != nil {
		h.Close()
		return nil, fmt.Errorf("cannot enable autorelay; autonat failed to start: %v", err)
	}
	h.SetAutoNat(autonat)

	// start the host background tasks
	h.Start()

	var ho host.Host
	ho = h
	if router != nil {
		ho = routed.Wrap(h, router)
	}
	if ar != nil {
		//创建自动中继Host
		return autorelay.NewAutoRelayHost(ho, ar), nil
	}
	return ho, nil
}

// Option is a libp2p config option that can be given to the libp2p constructor
// (`libp2p.New`).
type Option func(cfg *Config) error

// Apply applies the given options to the config, returning the first error
// encountered (if any). 应用配置
func (cfg *Config) Apply(opts ...Option) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(cfg); err != nil {
			return err
		}
	}
	return nil
}
