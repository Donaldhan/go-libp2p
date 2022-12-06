package client

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/record"
	pbv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/pb"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/proto"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/util"

	ma "github.com/multiformats/go-multiaddr"
)

var ReserveTimeout = time.Minute

// Reservation is a struct carrying information about a relay/v2 slot reservation.
//
//	relay/v2槽的预留服务
type Reservation struct {
	// Expiration is the expiration time of the reservation 过期时间
	Expiration time.Time
	// Addrs contains the vouched public addresses of the reserving peer, which can be
	// announced to the network peer的赌博地址
	Addrs []ma.Multiaddr

	// LimitDuration is the time limit for which the relay will keep a relayed connection
	// open. If 0, there is no limit. 保持中继连接的限制时间
	LimitDuration time.Duration
	// LimitData is the number of bytes that the relay will relay in each direction before
	// resetting a relayed connection. 在重置连接前的中继可以发送的字节数
	LimitData uint64

	// Voucher is a signed reservation voucher provided by the relay peer中继器信息
	Voucher *proto.ReservationVoucher
}

// Reserve reserves a slot in a relay and returns the reservation information.
// Clients must reserve slots in order for the relay to relay connections to them.
// 从中继器中预约一个服务，先占坑，染回预约信息，客户端必须按中继器连接的顺序预约slot
func Reserve(ctx context.Context, h host.Host, ai peer.AddrInfo) (*Reservation, error) {
	if len(ai.Addrs) > 0 {
		//添加中继到host的Peerstore
		h.Peerstore().AddAddrs(ai.ID, ai.Addrs, peerstore.TempAddrTTL)
	}
	//创建hop协议流
	s, err := h.NewStream(ctx, ai.ID, proto.ProtoIDv2Hop)
	if err != nil {
		return nil, err
	}
	defer s.Close()

	rd := util.NewDelimitedReader(s, maxMessageSize)
	wr := util.NewDelimitedWriter(s)
	defer rd.Close()

	var msg pbv2.HopMessage                   //hop消息
	msg.Type = pbv2.HopMessage_RESERVE.Enum() //预约中继slot

	s.SetDeadline(time.Now().Add(ReserveTimeout)) //设置预约超时时间

	if err := wr.WriteMsg(&msg); err != nil { //写预约服务消息
		s.Reset()
		return nil, fmt.Errorf("error writing reservation message: %w", err)
	}

	msg.Reset()

	if err := rd.ReadMsg(&msg); err != nil {
		s.Reset()
		return nil, fmt.Errorf("error reading reservation response message: %w", err)
	}

	if msg.GetType() != pbv2.HopMessage_STATUS { //非法相应 relay response
		return nil, fmt.Errorf("unexpected relay response: not a status message (%d)", msg.GetType())
	}

	if status := msg.GetStatus(); status != pbv2.Status_OK { //预约失败
		return nil, fmt.Errorf("reservation failed: %s (%d)", pbv2.Status_name[int32(status)], status)
	}
	//获取预约凭证
	rsvp := msg.GetReservation()
	if rsvp == nil {
		return nil, fmt.Errorf("missing reservation info")
	}

	result := &Reservation{}
	result.Expiration = time.Unix(int64(rsvp.GetExpire()), 0)
	if result.Expiration.Before(time.Now()) { //预约已过期
		return nil, fmt.Errorf("received reservation with expiration date in the past: %s", result.Expiration)
	}

	addrs := rsvp.GetAddrs() //预约地址
	result.Addrs = make([]ma.Multiaddr, 0, len(addrs))
	for _, ab := range addrs {
		a, err := ma.NewMultiaddrBytes(ab) //从字节恢复多播地址
		if err != nil {
			log.Warnf("ignoring unparsable relay address: %s", err)
			continue
		}
		result.Addrs = append(result.Addrs, a)
	}
	//预约凭证
	voucherBytes := rsvp.GetVoucher()
	if voucherBytes != nil {
		//???
		_, rec, err := record.ConsumeEnvelope(voucherBytes, proto.RecordDomain)
		if err != nil {
			return nil, fmt.Errorf("error consuming voucher envelope: %w", err)
		}

		voucher, ok := rec.(*proto.ReservationVoucher)
		if !ok {
			return nil, fmt.Errorf("unexpected voucher record type: %+T", rec)
		}
		result.Voucher = voucher
	}

	limit := msg.GetLimit() //凭证使用有效期
	if limit != nil {
		result.LimitDuration = time.Duration(limit.GetDuration()) * time.Second
		result.LimitData = limit.GetData()
	}

	return result, nil
}
