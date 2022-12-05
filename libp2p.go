package libp2p

import (
	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/host"
)

// Config describes a set of settings for a libp2p node.
type Config = config.Config

// Option is a libp2p config option that can be given to the libp2p constructor
// (`libp2p.New`).
type Option = config.Option

// ChainOptions chains multiple options into a single option.
func ChainOptions(opts ...Option) Option {
	return func(cfg *Config) error {
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
}

// New constructs a new libp2p node with the given options, falling back on
// reasonable defaults. The defaults are:
// 使用给定的选项创建一个libp2p 节点
// - If no transport and listen addresses are provided, the node listens to
// the multiaddresses "/ip4/0.0.0.0/tcp/0" and "/ip6/::/tcp/0";
// 如果没有选项提供，则将会在"/ip4/0.0.0.0/tcp/0" and "/ip6/::/tcp/0"多播地址上监听
// - If no transport options are provided, the node uses TCP, websocket and QUIC
// transport protocols;
// 如果没有transport选项，节点将会使用TCP, websocket and QUIC传输协议
// - If no multiplexer configuration is provided, the node is configured by
// default to use the "yamux/1.0.0" and "mplux/6.7.0" stream connection
// multiplexers;
// 如果没有多路复用器配置，节点将会使用"yamux/1.0.0" and "mplux/6.7.0流连接
// - If no security transport is provided, the host uses the go-libp2p's noise
// and/or tls encrypted transport to encrypt all traffic;
// 如果没有安全transport， 默认使用 go-libp2p' noise 加密transport
// - If no peer identity is provided, it generates a random RSA 2048 key-pair
// and derives a new identity from it;
// 若果没有peer identity将会产生一个RSA 2048的key对，表示当前节点
// - If no peerstore is provided, the host is initialized with an empty
// peerstore.
// peerstore没有提供，将会初始化一个空的peerstore
// To stop/shutdown the returned libp2p node, the user needs to cancel the passed context and call `Close` on the returned Host.
func New(opts ...Option) (host.Host, error) {
	return NewWithoutDefaults(append(opts, FallbackDefaults)...)
}

// NewWithoutDefaults constructs a new libp2p node with the given options but
// *without* falling back on reasonable defaults.
//
// Warning: This function should not be considered a stable interface. We may
// choose to add required services at any time and, by using this function, you
// opt-out of any defaults we may provide.
func NewWithoutDefaults(opts ...Option) (host.Host, error) {
	var cfg Config
	if err := cfg.Apply(opts...); err != nil {
		return nil, err
	}
	//新建peer node
	return cfg.NewNode()
}
