package pinger

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Report struct {
	Text           string
	RequestMessage *EchoMessage
	ReplyMessage   *EchoMessage
}

type EchoMessage struct {
	IP        net.IP
	ID        uint16
	Seq       uint16
	Timestamp time.Time
}

func NewEchoMessage(ip net.IP, seq uint16) *EchoMessage {
	return &EchoMessage{
		ip,
		makeEchoRequestID(binary.BigEndian.Uint32(ip)),
		seq,
		time.Now(),
	}
}

func BuildEchoMessageFromResponse(r *icmp.Message) *EchoMessage {
	body := r.Body.(*icmp.Echo)
	ip, ts := parsePayload(body.Data)
	if ip == nil {
		return nil
	}

	echo := &EchoMessage{
		ip,
		uint16(body.ID),
		uint16(body.Seq),
		ts,
	}

	if echo.ID != makeEchoRequestID(binary.BigEndian.Uint32(ip)) {
		return nil
	}

	return echo
}

func (r *EchoMessage) Next() *EchoMessage {
	nextseq := r.Seq + 1
	return NewEchoMessage(r.IP, nextseq)
}

func (r *EchoMessage) ICMPMessage() *icmp.Message {
	payload := encodePayload(r.IP, time.Now())

	return &icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: int(r.ID), Seq: int(r.Seq),
			Data: payload,
		},
	}
}

func (r *EchoMessage) String() string {
	return fmt.Sprintf("%v:%v:%v", r.IP.String(), r.ID, r.Seq)
}

func makeEchoRequestID(i uint32) uint16 {
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
