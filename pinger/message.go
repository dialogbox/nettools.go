package pinger

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type EchoRequest struct {
	IP          net.IP
	Seq         uint16
	RequestTime time.Time
	SentTime    time.Time
}

func NewEchoRequest(ip net.IP, seq uint16) *EchoRequest {
	return &EchoRequest{
		IP:          ip,
		Seq:         seq,
		RequestTime: time.Now(),
	}
}

func (r *EchoRequest) Sent() bool {
	return !r.SentTime.IsZero()
}

func (r *EchoRequest) ICMPMessage() *icmp.Message {
	payload := encodePayload(r.IP, time.Now())

	return &icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: int(getEchoID(r.IP)), Seq: int(r.Seq),
			Data: payload,
		},
	}
}

func (r *EchoRequest) Next() *EchoRequest {
	nextseq := r.Seq + 1
	return NewEchoRequest(r.IP, nextseq)
}

func (r *EchoRequest) HashKey() string {
	return fmt.Sprintf("%v:%v", r.IP.String(), r.Seq)
}

func (r *EchoRequest) String() string {
	return fmt.Sprintf("IP: %v, Seq: %v, Sent: %v", r.IP.String(), r.Seq, r.SentTime)
}

type EchoReply struct {
	IP           net.IP
	ID           uint16
	Seq          uint16
	SentTime     time.Time
	ReceivedTime time.Time
}

func NewEchoReplyFromResponse(r *icmp.Message) *EchoReply {
	body := r.Body.(*icmp.Echo)
	ip, ts := parsePayload(body.Data)
	// Unknown payload. This packet might be sent by the other sender. Ignore it
	if ip == nil || uint16(body.ID) != getEchoID(ip) {
		return nil
	}

	echo := &EchoReply{
		ip,
		uint16(body.ID),
		uint16(body.Seq),
		ts,
		time.Now(),
	}

	return echo
}

func (r *EchoReply) HashKey() string {
	return fmt.Sprintf("%v:%v", r.IP.String(), r.Seq)
}

func (r *EchoReply) String() string {
	return fmt.Sprintf("IP: %v, ID: %v, Seq: %v, Sent: %v, Returned: %v", r.IP.String(), r.ID, r.Seq, r.SentTime, r.ReceivedTime)
}

type ReportEventType uint8

const (
	REPORT_EVENT_ECHO_RECEIVED ReportEventType = iota
	REPORT_EVENT_PACKET_LOSS
	REPORT_EVENT_UNKNOWN_REPLY
	REPORT_EVENT_DUPLICATED_REQUEST
)

type Report struct {
	Event          ReportEventType
	RequestMessage *EchoRequest
	ReplyMessage   *EchoReply
}

func (r *Report) ElapsedTime() time.Duration {
	switch r.Event {
	case REPORT_EVENT_ECHO_RECEIVED:
		return r.ReplyMessage.ReceivedTime.Sub(r.RequestMessage.RequestTime)
	case REPORT_EVENT_PACKET_LOSS:
		return time.Now().Sub(r.RequestMessage.RequestTime)
	default:
		return 0
	}
}

func (r *Report) String() string {
	switch r.Event {
	case REPORT_EVENT_ECHO_RECEIVED:
		return fmt.Sprintf("From %v: icmp_seq=%v timegap=%v", r.ReplyMessage.IP, r.ReplyMessage.Seq, r.ElapsedTime())
	case REPORT_EVENT_PACKET_LOSS:
		return fmt.Sprintf("Packet loss from %v: icmp_seq=%v timegap=%v", r.RequestMessage.IP, r.RequestMessage.Seq, r.ElapsedTime())
	case REPORT_EVENT_UNKNOWN_REPLY:
		return fmt.Sprintf("Unknown reply from %v: icmp_seq=%v", r.ReplyMessage.IP, r.ReplyMessage.Seq)
	case REPORT_EVENT_DUPLICATED_REQUEST:
		return fmt.Sprintf("Duplicated request to %v: icmp_seq=%v", r.ReplyMessage.IP, r.ReplyMessage.Seq)
	default:
		panic("Unknown report type")
	}
}

// Utils

func getEchoID(ip net.IP) uint16 {
	i := binary.BigEndian.Uint32(ip)
	return uint16((i >> 16) ^ ((i & 0xffff) * 7))
}

const ECHO_PAYLOAD_SIZE = 12

func parsePayload(payload []byte) (net.IP, time.Time) {
	if len(payload) != ECHO_PAYLOAD_SIZE {
		return nil, time.Now()
	}

	ts := time.Unix(0, int64(binary.BigEndian.Uint64(payload[4:12])))
	ip := net.IP(payload[0:4]).To4()

	if ip != nil {
		return net.IP(payload[0:4]).To4(), ts
	} else {
		return nil, time.Now()
	}

}

func encodePayload(ip net.IP, ts time.Time) []byte {
	payload := make([]byte, ECHO_PAYLOAD_SIZE)

	copy(payload[0:4], ip[:])
	binary.BigEndian.PutUint64(payload[4:12], uint64(ts.UnixNano()))

	return payload
}
