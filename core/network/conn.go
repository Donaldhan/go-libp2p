package network

import (
	"context"
	"io"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	ma "github.com/multiformats/go-multiaddr"
)

// Conn is a connection to a remote peer. It multiplexes streams. 到远端peer的连接，多路服用流
// Usually there is no need to use a Conn directly, but it may
// be useful to get information about the peer on the other side:
// 通常，不需要直接使用连接，用于获取对等对应peer信息非常
//
//	stream.Conn().RemotePeer()
type Conn interface {
	io.Closer

	ConnSecurity
	ConnMultiaddrs
	ConnStat
	ConnScoper

	// ID returns an identifier that uniquely identifies this Conn within this
	// host, during this run. Connection IDs may repeat across restarts.
	ID() string

	// NewStream constructs a new Stream over this conn. 创建流
	NewStream(context.Context) (Stream, error)

	// GetStreams returns all open streams over this conn.
	GetStreams() []Stream
}

// ConnectionState holds information about the connection.
type ConnectionState struct {
	// The stream multiplexer used on this connection (if any). For example: /yamux/1.0.0 多路服用协议
	StreamMultiplexer string
	// The security protocol used on this connection (if any). For example: /tls/1.0.0 安全协议
	Security string
	// the transport used on this connection. For example: tcp  transport 协议
	Transport string
}

// ConnSecurity is the interface that one can mix into a connection interface to
// give it the security methods. 连接Key安全接口
type ConnSecurity interface {
	// LocalPeer returns our peer ID
	LocalPeer() peer.ID

	// LocalPrivateKey returns our private key
	LocalPrivateKey() ic.PrivKey

	// RemotePeer returns the peer ID of the remote peer.
	RemotePeer() peer.ID

	// RemotePublicKey returns the public key of the remote peer.
	RemotePublicKey() ic.PubKey

	// ConnState returns information about the connection state.
	ConnState() ConnectionState
}

// ConnMultiaddrs is an interface mixin for connection types that provide multiaddr
// addresses for the endpoints.  终端的多播地址
type ConnMultiaddrs interface {
	// LocalMultiaddr returns the local Multiaddr associated
	// with this connection
	LocalMultiaddr() ma.Multiaddr

	// RemoteMultiaddr returns the remote Multiaddr associated
	// with this connection
	RemoteMultiaddr() ma.Multiaddr
}

// ConnStat is an interface mixin for connection types that provide connection statistics. 连接统计
type ConnStat interface {
	// Stat stores metadata pertaining to this conn.
	Stat() ConnStats
}

// ConnScoper is the interface that one can mix into a connection interface to give it a resource
// management scope
type ConnScoper interface {
	// Scope returns the user view of this connection's resource scope 资源SCOPe
	Scope() ConnScope
}
