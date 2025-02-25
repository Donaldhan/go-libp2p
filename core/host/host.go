// Package host provides the core Host interface for libp2p.
//
// Host represents a single libp2p node in a peer-to-peer network.
package host

import (
	"context"

	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/introspection"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"

	ma "github.com/multiformats/go-multiaddr"
)

// Host is an object participating in a p2p network, which
// implements protocols or provides services. It handles
// requests like a Server, and issues requests like a Client.
// It is called Host because it is both Server and Client (and Peer
// may be confusing).
type Host interface {
	// ID returns the (local) peer.ID associated with this Host peerId 唯一标识
	ID() peer.ID

	// Peerstore returns the Host's repository of Peer Addresses and Keys. peer地址和key存储
	Peerstore() peerstore.Peerstore

	// Returns the listen addresses of the Host 监听的地址
	Addrs() []ma.Multiaddr

	// Networks returns the Network interface of the Host
	Network() network.Network

	// Mux returns the Mux multiplexing incoming streams to protocol handlers 用于处理incoming流的多路复用器
	Mux() protocol.Switch

	// Connect ensures there is a connection between this host and the peer with
	// given peer.ID. Connect will absorb the addresses in pi into its internal
	// peerstore. If there is not an active connection, Connect will issue a
	// h.Network.Dial, and block until a connection is open, or an error is
	// returned. // TODO: Relay + NAT.
	// 确保当前host和peer的连接，如果没有激活的连接，连接将会发从一个Dail拨号，阻塞到连接建立
	Connect(ctx context.Context, pi peer.AddrInfo) error

	// SetStreamHandler sets the protocol handler on the Host's Mux.
	// This is equivalent to:
	//   host.Mux().SetHandler(proto, handler)
	// (Threadsafe)
	// 设置host的多路服用的，协议处理器
	SetStreamHandler(pid protocol.ID, handler network.StreamHandler)

	// SetStreamHandlerMatch sets the protocol handler on the Host's Mux
	// using a matching function for protocol selection.
	// 设置用于协议选择匹配协议处理器
	SetStreamHandlerMatch(protocol.ID, func(string) bool, network.StreamHandler)

	// RemoveStreamHandler removes a handler on the mux that was set by
	// SetStreamHandler 移除流处理器
	RemoveStreamHandler(pid protocol.ID)

	// NewStream opens a new stream to given peer p, and writes a p2p/protocol
	// header with given ProtocolID. If there is no connection to p, attempts
	// to create one. If ProtocolID is "", writes no header.
	// (Threadsafe) 打开到给定peer的流
	NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error)

	// Close shuts down the host, its Network, and services. 关闭节点
	Close() error

	// ConnManager returns this hosts connection manager 连接管理器
	ConnManager() connmgr.ConnManager

	// EventBus returns the hosts eventbus 事件bus
	EventBus() event.Bus
}

// IntrospectableHost is implemented by Host implementations that are
// introspectable, that is, that may have introspection capability.
type IntrospectableHost interface {
	// Introspector returns the introspector, or nil if one hasn't been
	// registered. With it, the call can register data providers, and can fetch
	// introspection data.
	Introspector() introspection.Introspector

	// IntrospectionEndpoint returns the introspection endpoint, or nil if one
	// hasn't been registered.
	IntrospectionEndpoint() introspection.Endpoint
}
