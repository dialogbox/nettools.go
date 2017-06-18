package pinger

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type EchoMessage struct {
	IP        net.IP
	ID        uint16
	Seq       uint16
	Timestamp time.Time
}

func NewEchoMessage(ip net.IP, seq uint16) *EchoMessage {
	id := hash32to16(binary.BigEndian.Uint32(ip[12:16]))

	return &EchoMessage{
		ip,
		id,
		seq,
		time.Now(),
	}
}

func BuildEchoMessageFromResponse(m *icmp.Message) *EchoMessage {
	echo := m.Body.(*icmp.Echo)
	ip, ts := parsePayload(echo.Data)

	return &EchoMessage{
		ip,
		uint16(echo.ID),
		uint16(echo.Seq),
		ts,
	}
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
