package network

import (
	ma "github.com/multiformats/go-multiaddr"
)

// Notifiee is an interface for an object wishing to receive
// notifications from a Network. 希望从网络中接受通知的Notifiee对象
type Notifiee interface {
	Listen(Network, ma.Multiaddr)      // called when network starts listening on an addr 开启监听
	ListenClose(Network, ma.Multiaddr) // called when network stops listening on an addr 关闭监听
	Connected(Network, Conn)           // called when a connection opened 连接
	Disconnected(Network, Conn)        // called when a connection closed 关闭连接
}

// NotifyBundle implements Notifiee by calling any of the functions set on it,
// and nop'ing if they are unset. This is the easy way to register for
// notifications.
type NotifyBundle struct {
	ListenF      func(Network, ma.Multiaddr)
	ListenCloseF func(Network, ma.Multiaddr)

	ConnectedF    func(Network, Conn)
	DisconnectedF func(Network, Conn)
}

var _ Notifiee = (*NotifyBundle)(nil)

// Listen calls ListenF if it is not null.
func (nb *NotifyBundle) Listen(n Network, a ma.Multiaddr) {
	if nb.ListenF != nil {
		nb.ListenF(n, a)
	}
}

// ListenClose calls ListenCloseF if it is not null.
func (nb *NotifyBundle) ListenClose(n Network, a ma.Multiaddr) {
	if nb.ListenCloseF != nil {
		nb.ListenCloseF(n, a)
	}
}

// Connected calls ConnectedF if it is not null.
func (nb *NotifyBundle) Connected(n Network, c Conn) {
	if nb.ConnectedF != nil {
		nb.ConnectedF(n, c)
	}
}

// Disconnected calls DisconnectedF if it is not null.
func (nb *NotifyBundle) Disconnected(n Network, c Conn) {
	if nb.DisconnectedF != nil {
		nb.DisconnectedF(n, c)
	}
}

// Global noop notifiee. Do not change.
var GlobalNoopNotifiee = &NoopNotifiee{}

type NoopNotifiee struct{}

var _ Notifiee = (*NoopNotifiee)(nil)

func (nn *NoopNotifiee) Connected(n Network, c Conn)              {}
func (nn *NoopNotifiee) Disconnected(n Network, c Conn)           {}
func (nn *NoopNotifiee) Listen(n Network, addr ma.Multiaddr)      {}
func (nn *NoopNotifiee) ListenClose(n Network, addr ma.Multiaddr) {}
