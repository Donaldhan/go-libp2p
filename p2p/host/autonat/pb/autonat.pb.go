// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: autonat.proto

package autonat_pb

import (
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
// proto dial 拨号协议
// https://github.com/libp2p/specs/blob/master/autonat/README.md
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type Message_MessageType int32

const (
	Message_DIAL          Message_MessageType = 0 //拨号消息
	Message_DIAL_RESPONSE Message_MessageType = 1 //拨号响应
)

var Message_MessageType_name = map[int32]string{
	0: "DIAL",  //拨号消息
	1: "DIAL_RESPONSE", //拨号响应
}

var Message_MessageType_value = map[string]int32{
	"DIAL":          0,  //拨号消息
	"DIAL_RESPONSE": 1, //拨号响应
}

func (x Message_MessageType) Enum() *Message_MessageType {
	p := new(Message_MessageType)
	*p = x
	return p
}

func (x Message_MessageType) String() string {
	return proto.EnumName(Message_MessageType_name, int32(x))
}

func (x *Message_MessageType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(Message_MessageType_value, data, "Message_MessageType")
	if err != nil {
		return err
	}
	*x = Message_MessageType(value)
	return nil
}

func (Message_MessageType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_a04e278ef61ac07a, []int{0, 0}
}

type Message_ResponseStatus int32

const (
	Message_OK               Message_ResponseStatus = 0  //拨号成功响应
	Message_E_DIAL_ERROR     Message_ResponseStatus = 100 //拨号响应拒绝
	Message_E_DIAL_REFUSED   Message_ResponseStatus = 101 //拨号响应重用
	Message_E_BAD_REQUEST    Message_ResponseStatus = 200 //拨号响应Bad请求
	Message_E_INTERNAL_ERROR Message_ResponseStatus = 300 //拨号响应内部错误
)

var Message_ResponseStatus_name = map[int32]string{
	0:   "OK",
	100: "E_DIAL_ERROR",
	101: "E_DIAL_REFUSED",
	200: "E_BAD_REQUEST",
	300: "E_INTERNAL_ERROR",
}

var Message_ResponseStatus_value = map[string]int32{
	"OK":               0,
	"E_DIAL_ERROR":     100,
	"E_DIAL_REFUSED":   101,
	"E_BAD_REQUEST":    200,
	"E_INTERNAL_ERROR": 300,
}

func (x Message_ResponseStatus) Enum() *Message_ResponseStatus {
	p := new(Message_ResponseStatus)
	*p = x
	return p
}

func (x Message_ResponseStatus) String() string {
	return proto.EnumName(Message_ResponseStatus_name, int32(x))
}

func (x *Message_ResponseStatus) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(Message_ResponseStatus_value, data, "Message_ResponseStatus")
	if err != nil {
		return err
	}
	*x = Message_ResponseStatus(value)
	return nil
}

func (Message_ResponseStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_a04e278ef61ac07a, []int{0, 1}
}
//Dail消息
type Message struct {
	Type                 *Message_MessageType  `protobuf:"varint,1,opt,name=type,enum=autonat.pb.Message_MessageType" json:"type,omitempty"`
	Dial                 *Message_Dial         `protobuf:"bytes,2,opt,name=dial" json:"dial,omitempty"`
	DialResponse         *Message_DialResponse `protobuf:"bytes,3,opt,name=dialResponse" json:"dialResponse,omitempty"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}
func (*Message) Descriptor() ([]byte, []int) {
	return fileDescriptor_a04e278ef61ac07a, []int{0}
}
func (m *Message) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Message) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Message.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Message) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message.Merge(m, src)
}
func (m *Message) XXX_Size() int {
	return m.Size()
}
func (m *Message) XXX_DiscardUnknown() {
	xxx_messageInfo_Message.DiscardUnknown(m)
}

var xxx_messageInfo_Message proto.InternalMessageInfo

func (m *Message) GetType() Message_MessageType {
	if m != nil && m.Type != nil {
		return *m.Type
	}
	return Message_DIAL
}

func (m *Message) GetDial() *Message_Dial {
	if m != nil {
		return m.Dial
	}
	return nil
}

func (m *Message) GetDialResponse() *Message_DialResponse {
	if m != nil {
		return m.DialResponse
	}
	return nil
}
//peer消息
type Message_PeerInfo struct {
	Id                   []byte   `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Addrs                [][]byte `protobuf:"bytes,2,rep,name=addrs" json:"addrs,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Message_PeerInfo) Reset()         { *m = Message_PeerInfo{} }
func (m *Message_PeerInfo) String() string { return proto.CompactTextString(m) }
func (*Message_PeerInfo) ProtoMessage()    {}
func (*Message_PeerInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_a04e278ef61ac07a, []int{0, 0}
}
func (m *Message_PeerInfo) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Message_PeerInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Message_PeerInfo.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Message_PeerInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message_PeerInfo.Merge(m, src)
}
func (m *Message_PeerInfo) XXX_Size() int {
	return m.Size()
}
func (m *Message_PeerInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_Message_PeerInfo.DiscardUnknown(m)
}

var xxx_messageInfo_Message_PeerInfo proto.InternalMessageInfo

func (m *Message_PeerInfo) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *Message_PeerInfo) GetAddrs() [][]byte {
	if m != nil {
		return m.Addrs
	}
	return nil
}
//拨号消息
type Message_Dial struct {
	Peer                 *Message_PeerInfo `protobuf:"bytes,1,opt,name=peer" json:"peer,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *Message_Dial) Reset()         { *m = Message_Dial{} }
func (m *Message_Dial) String() string { return proto.CompactTextString(m) }
func (*Message_Dial) ProtoMessage()    {}
func (*Message_Dial) Descriptor() ([]byte, []int) {
	return fileDescriptor_a04e278ef61ac07a, []int{0, 1}
}
func (m *Message_Dial) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Message_Dial) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Message_Dial.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Message_Dial) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message_Dial.Merge(m, src)
}
func (m *Message_Dial) XXX_Size() int {
	return m.Size()
}
func (m *Message_Dial) XXX_DiscardUnknown() {
	xxx_messageInfo_Message_Dial.DiscardUnknown(m)
}

var xxx_messageInfo_Message_Dial proto.InternalMessageInfo

func (m *Message_Dial) GetPeer() *Message_PeerInfo {
	if m != nil {
		return m.Peer
	}
	return nil
}
//拨号响应消息
type Message_DialResponse struct {
	Status               *Message_ResponseStatus `protobuf:"varint,1,opt,name=status,enum=autonat.pb.Message_ResponseStatus" json:"status,omitempty"`
	StatusText           *string                 `protobuf:"bytes,2,opt,name=statusText" json:"statusText,omitempty"`
	Addr                 []byte                  `protobuf:"bytes,3,opt,name=addr" json:"addr,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *Message_DialResponse) Reset()         { *m = Message_DialResponse{} }
func (m *Message_DialResponse) String() string { return proto.CompactTextString(m) }
func (*Message_DialResponse) ProtoMessage()    {}
func (*Message_DialResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_a04e278ef61ac07a, []int{0, 2}
}
func (m *Message_DialResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Message_DialResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Message_DialResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Message_DialResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message_DialResponse.Merge(m, src)
}
func (m *Message_DialResponse) XXX_Size() int {
	return m.Size()
}
func (m *Message_DialResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_Message_DialResponse.DiscardUnknown(m)
}

var xxx_messageInfo_Message_DialResponse proto.InternalMessageInfo

func (m *Message_DialResponse) GetStatus() Message_ResponseStatus {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return Message_OK
}

func (m *Message_DialResponse) GetStatusText() string {
	if m != nil && m.StatusText != nil {
		return *m.StatusText
	}
	return ""
}

func (m *Message_DialResponse) GetAddr() []byte {
	if m != nil {
		return m.Addr
	}
	return nil
}
//初始化拨号协议
func init() {
	proto.RegisterEnum("autonat.pb.Message_MessageType", Message_MessageType_name, Message_MessageType_value)
	proto.RegisterEnum("autonat.pb.Message_ResponseStatus", Message_ResponseStatus_name, Message_ResponseStatus_value)
	proto.RegisterType((*Message)(nil), "autonat.pb.Message")
	proto.RegisterType((*Message_PeerInfo)(nil), "autonat.pb.Message.PeerInfo")
	proto.RegisterType((*Message_Dial)(nil), "autonat.pb.Message.Dial")
	proto.RegisterType((*Message_DialResponse)(nil), "autonat.pb.Message.DialResponse")
}

func init() { proto.RegisterFile("autonat.proto", fileDescriptor_a04e278ef61ac07a) }

var fileDescriptor_a04e278ef61ac07a = []byte{
	// 372 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x90, 0xcf, 0x8a, 0xda, 0x50,
	0x14, 0xc6, 0xbd, 0x31, 0xb5, 0xf6, 0x18, 0xc3, 0xed, 0xa1, 0x85, 0x20, 0x25, 0x0d, 0x59, 0x49,
	0x29, 0x22, 0x76, 0x53, 0xba, 0x53, 0x72, 0x0b, 0xd2, 0x56, 0xed, 0x49, 0x5c, 0x87, 0x94, 0xdc,
	0x0e, 0x01, 0x31, 0x21, 0x89, 0x30, 0x6e, 0xe6, 0x89, 0x66, 0x3b, 0xef, 0xe0, 0x72, 0x1e, 0x61,
	0xf0, 0x49, 0x86, 0x5c, 0xa3, 0xa3, 0xe0, 0xac, 0xce, 0x1f, 0x7e, 0xdf, 0x39, 0x1f, 0x1f, 0x74,
	0xa3, 0x4d, 0x99, 0xae, 0xa3, 0x72, 0x90, 0xe5, 0x69, 0x99, 0x22, 0x9c, 0xc6, 0x7f, 0xee, 0x83,
	0x0e, 0x6f, 0xff, 0xc8, 0xa2, 0x88, 0x6e, 0x24, 0x7e, 0x03, 0xbd, 0xdc, 0x66, 0xd2, 0x62, 0x0e,
	0xeb, 0x9b, 0xa3, 0xcf, 0x83, 0x17, 0x6c, 0x50, 0x23, 0xc7, 0x1a, 0x6c, 0x33, 0x49, 0x0a, 0xc6,
	0xaf, 0xa0, 0xc7, 0x49, 0xb4, 0xb2, 0x34, 0x87, 0xf5, 0x3b, 0x23, 0xeb, 0x9a, 0xc8, 0x4b, 0xa2,
	0x15, 0x29, 0x0a, 0x3d, 0x30, 0xaa, 0x4a, 0xb2, 0xc8, 0xd2, 0x75, 0x21, 0xad, 0xa6, 0x52, 0x39,
	0xaf, 0xaa, 0x6a, 0x8e, 0x2e, 0x54, 0xbd, 0x21, 0xb4, 0x17, 0x52, 0xe6, 0xd3, 0xf5, 0xff, 0x14,
	0x4d, 0xd0, 0x92, 0x58, 0x59, 0x36, 0x48, 0x4b, 0x62, 0xfc, 0x00, 0x6f, 0xa2, 0x38, 0xce, 0x0b,
	0x4b, 0x73, 0x9a, 0x7d, 0x83, 0x0e, 0x43, 0xef, 0x3b, 0xe8, 0xd5, 0x3d, 0x1c, 0x82, 0x9e, 0x49,
	0x99, 0x2b, 0xbe, 0x33, 0xfa, 0x74, 0xed, 0xef, 0xf1, 0x32, 0x29, 0xb2, 0x77, 0x07, 0xc6, 0xb9,
	0x13, 0xfc, 0x01, 0xad, 0xa2, 0x8c, 0xca, 0x4d, 0x51, 0xc7, 0xe4, 0x5e, 0xbb, 0x71, 0xa4, 0x7d,
	0x45, 0x52, 0xad, 0x40, 0x1b, 0xe0, 0xd0, 0x05, 0xf2, 0xb6, 0x54, 0x89, 0xbd, 0xa3, 0xb3, 0x0d,
	0x22, 0xe8, 0x95, 0x5d, 0x95, 0x8a, 0x41, 0xaa, 0x77, 0xbf, 0x40, 0xe7, 0x2c, 0x74, 0x6c, 0x83,
	0xee, 0x4d, 0xc7, 0xbf, 0x79, 0x03, 0xdf, 0x43, 0xb7, 0xea, 0x42, 0x12, 0xfe, 0x62, 0x3e, 0xf3,
	0x05, 0x67, 0x6e, 0x02, 0xe6, 0xe5, 0x67, 0x6c, 0x81, 0x36, 0xff, 0xc5, 0x1b, 0xc8, 0xc1, 0x10,
	0xa1, 0xc2, 0x05, 0xd1, 0x9c, 0x78, 0x8c, 0x08, 0x66, 0xbd, 0x21, 0xf1, 0x73, 0xe9, 0x0b, 0x8f,
	0x4b, 0x44, 0xe8, 0x8a, 0x70, 0x32, 0xf6, 0x42, 0x12, 0x7f, 0x97, 0xc2, 0x0f, 0xf8, 0x8e, 0xe1,
	0x47, 0xe0, 0x22, 0x9c, 0xce, 0x02, 0x41, 0xb3, 0x93, 0xfa, 0x5e, 0x9b, 0x18, 0xbb, 0xbd, 0xcd,
	0x1e, 0xf7, 0x36, 0x7b, 0xda, 0xdb, 0xec, 0x39, 0x00, 0x00, 0xff, 0xff, 0x8e, 0xe2, 0x93, 0x4e,
	0x61, 0x02, 0x00, 0x00,
}

func (m *Message) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Message) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Message) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.DialResponse != nil {
		{
			size, err := m.DialResponse.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintAutonat(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x1a
	}
	if m.Dial != nil {
		{
			size, err := m.Dial.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintAutonat(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if m.Type != nil {
		i = encodeVarintAutonat(dAtA, i, uint64(*m.Type))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *Message_PeerInfo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Message_PeerInfo) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Message_PeerInfo) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Addrs) > 0 {
		for iNdEx := len(m.Addrs) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Addrs[iNdEx])
			copy(dAtA[i:], m.Addrs[iNdEx])
			i = encodeVarintAutonat(dAtA, i, uint64(len(m.Addrs[iNdEx])))
			i--
			dAtA[i] = 0x12
		}
	}
	if m.Id != nil {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintAutonat(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *Message_Dial) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Message_Dial) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Message_Dial) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.Peer != nil {
		{
			size, err := m.Peer.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintAutonat(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *Message_DialResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Message_DialResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Message_DialResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.Addr != nil {
		i -= len(m.Addr)
		copy(dAtA[i:], m.Addr)
		i = encodeVarintAutonat(dAtA, i, uint64(len(m.Addr)))
		i--
		dAtA[i] = 0x1a
	}
	if m.StatusText != nil {
		i -= len(*m.StatusText)
		copy(dAtA[i:], *m.StatusText)
		i = encodeVarintAutonat(dAtA, i, uint64(len(*m.StatusText)))
		i--
		dAtA[i] = 0x12
	}
	if m.Status != nil {
		i = encodeVarintAutonat(dAtA, i, uint64(*m.Status))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintAutonat(dAtA []byte, offset int, v uint64) int {
	offset -= sovAutonat(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Message) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Type != nil {
		n += 1 + sovAutonat(uint64(*m.Type))
	}
	if m.Dial != nil {
		l = m.Dial.Size()
		n += 1 + l + sovAutonat(uint64(l))
	}
	if m.DialResponse != nil {
		l = m.DialResponse.Size()
		n += 1 + l + sovAutonat(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *Message_PeerInfo) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Id != nil {
		l = len(m.Id)
		n += 1 + l + sovAutonat(uint64(l))
	}
	if len(m.Addrs) > 0 {
		for _, b := range m.Addrs {
			l = len(b)
			n += 1 + l + sovAutonat(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *Message_Dial) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Peer != nil {
		l = m.Peer.Size()
		n += 1 + l + sovAutonat(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *Message_DialResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Status != nil {
		n += 1 + sovAutonat(uint64(*m.Status))
	}
	if m.StatusText != nil {
		l = len(*m.StatusText)
		n += 1 + l + sovAutonat(uint64(l))
	}
	if m.Addr != nil {
		l = len(m.Addr)
		n += 1 + l + sovAutonat(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovAutonat(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozAutonat(x uint64) (n int) {
	return sovAutonat(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Message) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAutonat
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Message: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Message: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Type", wireType)
			}
			var v Message_MessageType
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAutonat
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= Message_MessageType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Type = &v
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Dial", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAutonat
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthAutonat
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthAutonat
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Dial == nil {
				m.Dial = &Message_Dial{}
			}
			if err := m.Dial.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DialResponse", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAutonat
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthAutonat
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthAutonat
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.DialResponse == nil {
				m.DialResponse = &Message_DialResponse{}
			}
			if err := m.DialResponse.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAutonat(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthAutonat
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthAutonat
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Message_PeerInfo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAutonat
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: PeerInfo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PeerInfo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAutonat
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthAutonat
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthAutonat
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = append(m.Id[:0], dAtA[iNdEx:postIndex]...)
			if m.Id == nil {
				m.Id = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Addrs", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAutonat
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthAutonat
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthAutonat
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Addrs = append(m.Addrs, make([]byte, postIndex-iNdEx))
			copy(m.Addrs[len(m.Addrs)-1], dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAutonat(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthAutonat
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthAutonat
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Message_Dial) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAutonat
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Dial: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Dial: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Peer", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAutonat
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthAutonat
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthAutonat
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Peer == nil {
				m.Peer = &Message_PeerInfo{}
			}
			if err := m.Peer.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAutonat(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthAutonat
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthAutonat
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Message_DialResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAutonat
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DialResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DialResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			var v Message_ResponseStatus
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAutonat
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= Message_ResponseStatus(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Status = &v
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field StatusText", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAutonat
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthAutonat
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthAutonat
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			s := string(dAtA[iNdEx:postIndex])
			m.StatusText = &s
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Addr", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAutonat
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthAutonat
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthAutonat
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Addr = append(m.Addr[:0], dAtA[iNdEx:postIndex]...)
			if m.Addr == nil {
				m.Addr = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAutonat(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthAutonat
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthAutonat
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipAutonat(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowAutonat
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowAutonat
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowAutonat
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthAutonat
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupAutonat
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthAutonat
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthAutonat        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowAutonat          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupAutonat = fmt.Errorf("proto: unexpected end of group")
)
