package autorelay

import (
	"github.com/libp2p/go-libp2p/core/host"
)

// auto中继host
type AutoRelayHost struct {
	host.Host
	ar *AutoRelay
}

// 关闭host
func (h *AutoRelayHost) Close() error {
	_ = h.ar.Close()
	return h.Host.Close()
}

func NewAutoRelayHost(h host.Host, ar *AutoRelay) *AutoRelayHost {
	return &AutoRelayHost{Host: h, ar: ar}
}
