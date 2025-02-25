package autorelay

import (
	"context"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/benbjohnson/clock"
)

type config struct {
	clock      clock.Clock
	peerSource func(ctx context.Context, num int) <-chan peer.AddrInfo
	// minimum interval used to call the peerSource callback
	minInterval time.Duration
	// see WithMinCandidates 最小候选节点数
	minCandidates int
	// see WithMaxCandidates 最大候选节点数
	maxCandidates int
	// Delay until we obtain reservations with relays, if we have less than minCandidates candidates. 延迟boot时间，直到我们获取预留的中继器数量
	// See WithBootDelay.
	bootDelay time.Duration
	// backoff is the time we wait after failing to obtain a reservation with a candidate 退避时间， 在我们获取候选的reservation失败时，等待的时间
	backoff time.Duration
	// Number of relays we strive to obtain a reservation with.  需要中继器数量
	desiredRelays int
	// see WithMaxCandidateAge 最大候选时间
	maxCandidateAge  time.Duration
	setMinCandidates bool
	enableCircuitV1  bool
}

// 默认配置
var defaultConfig = config{
	clock:           clock.New(),
	minCandidates:   4,
	maxCandidates:   20,
	bootDelay:       3 * time.Minute,
	backoff:         time.Hour,
	desiredRelays:   2,
	maxCandidateAge: 30 * time.Minute,
}

var (
	errAlreadyHavePeerSource = errors.New("can only use a single WithPeerSource or WithStaticRelays")
)

type Option func(*config) error

func WithStaticRelays(static []peer.AddrInfo) Option {
	return func(c *config) error {
		if c.peerSource != nil {
			return errAlreadyHavePeerSource
		}

		WithPeerSource(func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
			if len(static) < numPeers {
				numPeers = len(static)
			}
			c := make(chan peer.AddrInfo, numPeers)
			defer close(c)

			for i := 0; i < numPeers; i++ {
				c <- static[i]
			}
			return c
		}, 30*time.Second)(c)
		WithMinCandidates(len(static))(c)
		WithMaxCandidates(len(static))(c)
		WithNumRelays(len(static))(c)

		return nil
	}
}

// 配置peersource
// WithPeerSource defines a callback for AutoRelay to query for more relay candidates.
// AutoRelay will call this function when it needs new candidates is connected to the desired number of
// relays, and it has enough candidates (in case we get disconnected from one of the relays).
// Implementations must send *at most* numPeers, and close the channel when they don't intend to provide
// any more peers.
// AutoRelay will not call the callback again until the channel is closed.
// Implementations should send new peers, but may send peers they sent before. AutoRelay implements
// a per-peer backoff (see WithBackoff).
// minInterval is the minimum interval this callback is called with, even if AutoRelay needs new candidates.
// The context.Context passed MAY be canceled when AutoRelay feels satisfied, it will be canceled when the node is shutting down.
// If the channel is canceled you MUST close the output channel at some point.
func WithPeerSource(f func(ctx context.Context, numPeers int) <-chan peer.AddrInfo, minInterval time.Duration) Option {
	return func(c *config) error {
		if c.peerSource != nil {
			return errAlreadyHavePeerSource
		}
		c.peerSource = f
		c.minInterval = minInterval
		return nil
	}
}

// WithNumRelays sets the number of relays we strive to obtain reservations with.
func WithNumRelays(n int) Option {
	return func(c *config) error {
		c.desiredRelays = n
		return nil
	}
}

// WithMaxCandidates sets the number of relay candidates that we buffer. 最大候选者
func WithMaxCandidates(n int) Option {
	return func(c *config) error {
		c.maxCandidates = n
		if c.minCandidates > n {
			c.minCandidates = n
		}
		return nil
	}
}

// WithMinCandidates sets the minimum number of relay candidates we collect before to get a reservation 最小候选者
// with any of them (unless we've been running for longer than the boot delay).
// This is to make sure that we don't just randomly connect to the first candidate that we discover.
func WithMinCandidates(n int) Option {
	return func(c *config) error {
		if n > c.maxCandidates {
			n = c.maxCandidates
		}
		c.minCandidates = n
		c.setMinCandidates = true
		return nil
	}
}

// WithBootDelay set the boot delay for finding relays. 启动延迟
// We won't attempt any reservation if we've have less than a minimum number of candidates.
// This prevents us to connect to the "first best" relay, and allows us to carefully select the relay.
// However, in case we haven't found enough relays after the boot delay, we use what we have.
func WithBootDelay(d time.Duration) Option {
	return func(c *config) error {
		c.bootDelay = d
		return nil
	}
}

// WithBackoff sets the time we wait after failing to obtain a reservation with a candidate.
func WithBackoff(d time.Duration) Option {
	return func(c *config) error {
		c.backoff = d
		return nil
	}
}

// WithCircuitV1Support enables support for circuit v1 relays. 是否支持circuit v1 relays
func WithCircuitV1Support() Option {
	return func(c *config) error {
		c.enableCircuitV1 = true
		return nil
	}
}

// WithMaxCandidateAge sets the maximum age of a candidate. 候选者最大age
// When we are connected to the desired number of relays, we don't ask the peer source for new candidates.
// This can lead to AutoRelay's candidate list becoming outdated, and means we won't be able
// to quickly establish a new relay connection if our existing connection breaks, if all the candidates
// have become stale.
func WithMaxCandidateAge(d time.Duration) Option {
	return func(c *config) error {
		c.maxCandidateAge = d
		return nil
	}
}

func WithClock(cl clock.Clock) Option {
	return func(c *config) error {
		c.clock = cl
		return nil
	}
}
