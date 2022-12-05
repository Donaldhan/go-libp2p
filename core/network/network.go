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
	// node is unknown.
	ReachabilityUnknown Reachability = iota

	// ReachabilityPublic indicates that the node is reachable from the
	// public internet.
	ReachabilityPublic

	// ReachabilityPrivate indicates that the node is not reachable from the
	// public internet.
	//
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
	// NumStreams is the number of streams on the connection.
	NumStreams int
}

// Stats stores metadata pertaining to a given Stream / Conn.
type Stats struct {
	// Direction specifies whether this is an inbound or an outbound connection.
	Direction Direction
	// Opened is the timestamp when this connection was opened.
	Opened time.Time
	// Transient indicates that this connection is transient and may be closed soon.
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
	Dialer
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

	// LocalPeer returns the local peer associated with this network
	LocalPeer() peer.ID

	// DialPeer establishes a connection to a given peer
	DialPeer(context.Context, peer.ID) (Conn, error)

	// ClosePeer closes the connection to a given peer
	ClosePeer(peer.ID) error

	// Connectedness returns a state signaling connection capabilities
	Connectedness(peer.ID) Connectedness

	// Peers returns the peers connected
	Peers() []peer.ID

	// Conns returns the connections in this Network
	Conns() []Conn

	// ConnsToPeer returns the connections in this Network for given peer.
	ConnsToPeer(p peer.ID) []Conn

	// Notify/StopNotify register and unregister a notifiee for signals
	Notify(Notifiee)
	StopNotify(Notifiee)
}
