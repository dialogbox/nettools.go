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
	"encoding/binary"
	"os"
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

func (p *Pinger) Seq() uint16 {
	for {
		seq := atomic.AddUint32(&p.seq, 1)

		if seq <= 65535 {
			return uint16(seq)
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
	ipaddr, err := LookupIPAddr(host)
	if err != nil {
		panic(err)
	}

	wm := NewEchoRequest(ipaddr)

	wb, err := wm.Marshal(ProtocolICMP)
	if err != nil {
		log.Fatal(err)
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

// Request Registry

func hash32to16(i uint32) uint16 {
	return uint16((i >> 16) ^ ((i & 0xffff) * 7))
}

func NewEchoRequest(ip net.IP) *icmp.Echo {
	id := hash32to16(uint32(os.Getpid()))

	return &icmp.Echo{
		ID: int(id), Seq: 1,
		Data: ip[12:16],
	}
}


func NewEchoRequest2(ip net.IP, seq uint16) *icmp.Echo {
	id := hash32to16(binary.BigEndian.Uint32(ip[12:16]))
	return &icmp.Echo{
		ID: int(id), Seq: int(seq),
		Data: ip[12:16],
	}
}
