package event

import (
	"github.com/libp2p/go-libp2p/core/network"
)

// EvtLocalReachabilityChanged is an event struct to be emitted when the local's
// node reachability changes state.
// 当本地节点可达状态变更时，emit event
// This event is usually emitted by the AutoNAT subsystem.
type EvtLocalReachabilityChanged struct {
	Reachability network.Reachability
}
