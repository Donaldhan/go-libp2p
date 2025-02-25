package event

import (
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// EvtPeerConnectednessChanged should be emitted every time the "connectedness" to a
// given peer changes. Specifically, this event is emitted in the following
// cases:
//
//   - Connectedness = Connected: Every time we transition from having no
//     connections to a peer to having at least one connection to the peer. 连接状态：从无连接到至少有一个到peer的连接
//   - Connectedness = NotConnected: Every time we transition from having at least
//     one connection to a peer to having no connections to the peer. 无连接状态：从至少一个连接到无连接
//
// Additional connectedness states may be added in the future. This list should
// not be considered exhaustive.
//
// Take note:
//
//   - It's possible to have _multiple_ connections to a given peer.
//   - Both libp2p and networks are asynchronous. 由于网络是异步的，可能存在多个到peer的连接
//
// This means that all of the following situations are possible:
//
// A connection is cut and is re-established: 连接重新建立
//
//   - Peer A observes a transition from Connected -> NotConnected -> Connected 观察的连接阐述
//   - Peer B observes a transition from Connected -> NotConnected -> Connected
//
// Explanation: Both peers observe the connection die. This is the "nice" case.
//
// A connection is cut and is re-established.
//
//   - Peer A observes a transition from Connected -> NotConnected -> Connected.
//   - Peer B observes no transition.
//
// Explanation: Peer A re-establishes the dead connection. Peer B observes the
// new connection form before it observes the old connection die.
//
// A connection is cut:
//
//   - Peer A observes no transition.
//   - Peer B observes no transition.
//
// Explanation: There were two connections and one was cut. This connection
// might have been in active use but neither peer will observe a change in
// "connectedness". Peers should always make sure to re-try network requests.
// 节点连接状态
type EvtPeerConnectednessChanged struct {
	// Peer is the remote peer who's connectedness has changed.
	Peer peer.ID
	// Connectedness is the new connectedness state.
	Connectedness network.Connectedness
}
