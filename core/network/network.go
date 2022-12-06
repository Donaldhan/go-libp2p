// Package network provides core networking abstractions for libp2p.
//
// The network package provides the high-level Network interface for interacting
// with other libp2p peers, which is the primary public API for initiating and
// accepting connections to remote peers.
package network

import (
	"context"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"

	ma "github.com/multiformats/go-multiaddr"
)

// MessageSizeMax is a soft (recommended) maximum for network messages.
// One can write more, as the interface is a stream. But it is useful
// to bunch it up into multiple read/writes when the whole message is
// a single, large serialized object.
const MessageSizeMax = 1 << 22 // 4 MB

// Direction represents which peer in a stream initiated a connection.
type Direction int

const (
	// DirUnknown is the default direction.
	DirUnknown Direction = iota
	// DirInbound is for when the remote peer initiated a connection.
	DirInbound
	// DirOutbound is for when the local peer initiated a connection.
	DirOutbound
)

func (d Direction) String() string {
	str := [...]string{"Unknown", "Inbound", "Outbound"}
	if d < 0 || int(d) >= len(str) {
		return "(unrecognized)"
	}
	return str[d]
}

// Connectedness signals the capacity for a connection with a given node.
// It is used to signal to services and other peers whether a node is reachable.
type Connectedness int

const (
	// NotConnected means no connection to peer, and no extra information (default)
	NotConnected Connectedness = iota

	// Connected means has an open, live connection to peer
	Connected

	// CanConnect means recently connected to peer, terminated gracefully
	CanConnect

	// CannotConnect means recently attempted connecting but failed to connect.
	// (should signal "made effort, failed")
	CannotConnect
)

func (c Connectedness) String() string {
	str := [...]string{"NotConnected", "Connected", "CanConnect", "CannotConnect"}
	if c < 0 || int(c) >= len(str) {
		return "(unrecognized)"
	}
	return str[c]
}

// Reachability indicates how reachable a node is.
type Reachability int

const (
	// ReachabilityUnknown indicates that the reachability status of the
	// node is unknown. 节点未知
	ReachabilityUnknown Reachability = iota

	// ReachabilityPublic indicates that the node is reachable from the
	// public internet.节点公网可达
	ReachabilityPublic

	// ReachabilityPrivate indicates that the node is not reachable from the
	// public internet.
	// 节点公网不可达， 但是节点仍有可能通过中继器可达
	// NOTE: This node may _still_ be reachable via relays.
	ReachabilityPrivate
)

func (r Reachability) String() string {
	str := [...]string{"Unknown", "Public", "Private"}
	if r < 0 || int(r) >= len(str) {
		return "(unrecognized)"
	}
	return str[r]
}

// ConnStats stores metadata pertaining to a given Conn.
type ConnStats struct {
	Stats
	// NumStreams is the number of streams on the connection. 在当前连接上的流数量
	NumStreams int
}

// Stats stores metadata pertaining to a given Stream / Conn.
type Stats struct {
	// Direction specifies whether this is an inbound or an outbound connection.
	Direction Direction
	// Opened is the timestamp when this connection was opened. 连接开启时间
	Opened time.Time
	// Transient indicates that this connection is transient and may be closed soon. 通道瞬态标志，表示连接即将关闭
	Transient bool
	// Extra stores additional metadata about this connection.
	Extra map[interface{}]interface{}
}

// StreamHandler is the type of function used to listen for
// streams opened by the remote side.
type StreamHandler func(Stream)

// Network is the interface used to connect to the outside world.
// It dials and listens for connections. it uses a Swarm to pool
// connections (see swarm pkg, and peerstream.Swarm). Connections
// are encrypted with a TLS-like protocol.
type Network interface {
	Dialer //继承
	io.Closer

	// SetStreamHandler sets the handler for new streams opened by the
	// remote side. This operation is threadsafe. 设置流处理器
	SetStreamHandler(StreamHandler)

	// NewStream returns a new stream to given peer p. 新创建一个流
	// If there is no connection to p, attempts to create one.
	NewStream(context.Context, peer.ID) (Stream, error)

	// Listen tells the network to start listening on given multiaddrs. 监听多播地址
	Listen(...ma.Multiaddr) error

	// ListenAddresses returns a list of addresses at which this network listens. 获取监听的地址
	ListenAddresses() []ma.Multiaddr

	// InterfaceListenAddresses returns a list of addresses at which this network
	// listens. It expands "any interface" addresses (/ip4/0.0.0.0, /ip6/::) to
	// use the known local interfaces. 网络监听地址
	InterfaceListenAddresses() ([]ma.Multiaddr, error)

	// ResourceManager returns the ResourceManager associated with this network 网络资源管理器
	ResourceManager() ResourceManager
}

// Dialer represents a service that can dial out to peers 表示一个可以dial Peers的Dialer
// (this is usually just a Network, but other services may not need the whole
// stack, and thus it becomes easier to mock)
type Dialer interface {
	// Peerstore returns the internal peerstore peer存储
	// This is useful to tell the dialer about a new address for a peer.
	// Or use one of the public keys found out over the network.
	Peerstore() peerstore.Peerstore

	// LocalPeer returns the local peer associated with this network 本机节点地址
	LocalPeer() peer.ID

	// DialPeer establishes a connection to a given peer 拨号
	DialPeer(context.Context, peer.ID) (Conn, error)

	// ClosePeer closes the connection to a given peer 关闭给定节点的连接
	ClosePeer(peer.ID) error

	// Connectedness returns a state signaling connection capabilities 返回peer的连接状态
	Connectedness(peer.ID) Connectedness

	// Peers returns the peers connected 返回连接的peerID
	Peers() []peer.ID

	// Conns returns the connections in this Network 返回当前网络的连接
	Conns() []Conn

	// ConnsToPeer returns the connections in this Network for given peer. 返回给定peer所有网络连接
	ConnsToPeer(p peer.ID) []Conn

	// Notify/StopNotify register and unregister a notifiee for signals 注册或注销notifiee信号
	Notify(Notifiee)
	StopNotify(Notifiee)
}
