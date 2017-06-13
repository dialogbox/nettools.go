package nettools

import (
	"encoding/binary"
	"errors"
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"log"
	"net"
	"time"
)



const ProtocolICMP = 1
const ProtocolIPv6ICMP = 58

func LookupIPAddr(host string) (net.IP, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	for _, ip := range ips {
		return ip, nil
	}

	return nil, errors.New("no A or AAAA record")
}

type Pinger struct {
	Conn *icmp.PacketConn
}


func NewPinger(timeout time.Duration) (*Pinger, error) {
	c, err := icmp.ListenPacket("udp4", "")
	if err != nil {
		return nil, err
	}
	if err := c.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		panic(err)
	}

	return &Pinger{Conn: c}, nil
}

func (p *Pinger) Ping(ip net.IP, seq uint16) {
	r := NewEchoRequest(ip, seq)
	wm := r.GetPacket()

	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := p.Conn.WriteTo(wb, &net.UDPAddr{IP: ip}); err != nil {
		panic(err)
	}
}

func (p *Pinger) GetPacket() (string, *icmp.Message, error) {
	rb := make([]byte, 1500)
	n, peer, err := p.Conn.ReadFrom(rb)
	if err != nil {
		return "", nil, err
	}

	peerIP := peer.String()
	peerIP = peerIP[:len(peerIP)-2]

	rm, err := icmp.ParseMessage(ProtocolICMP, rb[:n])
	if err != nil {
		return peerIP, nil, err
	}

	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		return peerIP, rm, nil
	default:
		return peerIP, nil, fmt.Errorf("got %+v; want echo reply", rm)
	}
}

func (p *Pinger) Close() {
	p.Conn.Close()
}

type EchoRequest struct {
	IP net.IP
	Id uint16
	Seq uint16
	Timestamp time.Time

}

func hash32to16(i uint32) uint16 {
	return uint16((i >> 16) ^ ((i & 0xffff) * 7))
}

func NewEchoRequest(ip net.IP, seq uint16) *EchoRequest {
	id := hash32to16(binary.BigEndian.Uint32(ip[12:16]))

	return &EchoRequest{
		ip,
		id,
		seq,
		time.Now(),
	}
}

func BuildEchoRequestFromResponse(m *icmp.Message) *EchoRequest {
	echo := m.Body.(*icmp.Echo)
	ip, ts := parsePayload(echo.Data)

	return &EchoRequest{
		ip,
		uint16(echo.ID),
		uint16(echo.Seq),
		ts,
	}
}

func (r *EchoRequest) GetPacket() *icmp.Message {
	payload := encodePayload(r.IP, r.Timestamp)

	return &icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID:   int(r.Id), Seq: int(r.Seq),
			Data: payload,
		},
	}
}

func (r *EchoRequest) String() string {
	return fmt.Sprintf("%v:%v:%v", r.IP, r.Id, r.Seq)
}

func ResolveHostList(hosts []string) []net.IP {
	iplist := make([]net.IP, len(hosts))
	for i := range hosts {
		ip, err := LookupIPAddr(hosts[i])
		if err != nil {
			panic(err)
		}
		iplist[i] = ip
	}

	return iplist
}


func parsePayload(payload []byte) (net.IP, time.Time) {
	var ip [4]byte

	copy(ip[:], payload[0:4])

	ts := time.Unix(0, int64(binary.BigEndian.Uint64(payload[4:12])))

	return net.IP(ip[:]), ts
}

func encodePayload(ip net.IP, ts time.Time) []byte {
	payload := make([]byte, 12)

	copy(payload[0:4], ip[12:16])
	binary.BigEndian.PutUint64(payload[4:12], uint64(ts.UnixNano()))

	return payload
}

type RequestRegistry struct {
	timeout time.Duration
	db map[string]bool
}

func NewRequestRegistry(timeout time.Duration) *RequestRegistry {
	return &RequestRegistry{timeout, make(map[string]bool)}
}

func (registry *RequestRegistry) Regist(m *EchoRequest) bool {
	if registry.db[m.String()] {
		return false
	}
	registry.db[m.String()] = true

	return true
}