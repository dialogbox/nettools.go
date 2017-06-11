package nettools

import (
	"log"
	"net"
	"time"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"errors"
	"sync/atomic"
	"fmt"
	"math/rand"
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
	Id uint16
	seq uint32
}

func (p *Pinger) Seq() uint32 {
	for {
		seq := atomic.AddUint32(&p.seq, 1)

		if seq <= 65535 {
			return seq
		}
		atomic.SwapUint32(&p.seq, 0)
	}
}

func NewPinger() (*Pinger, error) {
	c, err := icmp.ListenPacket("udp4", "")
	if err != nil {
		return nil, err
	}
	if err := c.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		panic(err)
	}

	id := uint16(rand.Intn(65535))

	return &Pinger{Conn: c, Id: id }, nil
}

func (p *Pinger) Ping(host string) {
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: int(p.Id), Seq: int(p.Seq()),
			Data: []byte("P"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Fatal(err)
	}

	ipaddr, err := LookupIPAddr(host)
	if err != nil {
		panic(err)
	}

	if _, err := p.Conn.WriteTo(wb, &net.UDPAddr{IP: ipaddr}); err != nil {
		panic(err)
	}
}

func (p *Pinger) GetPacket() (*net.Addr, *icmp.Message, error) {
	rb := make([]byte, 1500)
	n, peer, err := p.Conn.ReadFrom(rb)
	if err != nil {
		return nil, nil, err
	}

	rm, err := icmp.ParseMessage(ProtocolICMP, rb[:n])
	if err != nil {
		return &peer, nil, err
	}

	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		return &peer, rm, nil
	default:
		return &peer, nil, fmt.Errorf("got %+v; want echo reply", rm)
	}
}

func (p *Pinger) Close() {
	p.Conn.Close()
}