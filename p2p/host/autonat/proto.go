package autonat

import (
	"github.com/libp2p/go-libp2p/core/peer"
	pb "github.com/libp2p/go-libp2p/p2p/host/autonat/pb"

	ma "github.com/multiformats/go-multiaddr"
)

// AutoNATProto identifies the autonat service protocol AutoNat协议
const AutoNATProto = "/libp2p/autonat/1.0.0"

// 创建拨号消息
func newDialMessage(pi peer.AddrInfo) *pb.Message {
	msg := new(pb.Message)
	msg.Type = pb.Message_DIAL.Enum() //拨号消息
	msg.Dial = new(pb.Message_Dial)
	msg.Dial.Peer = new(pb.Message_PeerInfo)            //拨号Peer信息
	msg.Dial.Peer.Id = []byte(pi.ID)                    //拨号PeerId
	msg.Dial.Peer.Addrs = make([][]byte, len(pi.Addrs)) //拨号peer地址
	for i, addr := range pi.Addrs {
		msg.Dial.Peer.Addrs[i] = addr.Bytes()
	}

	return msg
}

// 创建拨号成功响应消息
func newDialResponseOK(addr ma.Multiaddr) *pb.Message_DialResponse {
	dr := new(pb.Message_DialResponse)
	dr.Status = pb.Message_OK.Enum()
	dr.Addr = addr.Bytes()
	return dr
}

// 创建拨号错误响应消息
func newDialResponseError(status pb.Message_ResponseStatus, text string) *pb.Message_DialResponse {
	dr := new(pb.Message_DialResponse)
	dr.Status = status.Enum()
	dr.StatusText = &text
	return dr
}
