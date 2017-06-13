package nettools

import (
	"fmt"
	"golang.org/x/net/icmp"
	"testing"
	"time"
	"net"
)

func TestNewPinger(t *testing.T) {
	p, err := NewPinger(3 * time.Second)
	if err != nil {
		panic(err)
	}
	defer p.Close()
}

func TestPing(t *testing.T) {
	table := []string{
		"127.0.0.1",
		"www.google.com",
		"www.apple.com",
	}

	p, err := NewPinger(3 * time.Second)
	if err != nil {
		panic(err)
	}
	defer p.Close()

	for i := range table {
		ip, err := LookupIPAddr(table[i])
		if err != nil {
			panic(err)
		}
		p.Ping(ip, 1)
		peer, rm, err := p.GetPacket()
		if err != nil {
			panic(err)
		}
		e := rm.Body.(*icmp.Echo)

		addr, ts := parsePayload(e.Data)

		if peer != addr.String() {
			fmt.Printf("%v, %v, %v, %v, %v\n", peer, e.ID, e.Seq, addr, ts)
		}
	}
}

func TestResolveHostList(t *testing.T) {
	table := []string{
		"127.0.0.1",
		"www.google.com",
		"www.apple.com",
	}

	iplist := ResolveHostList(table)
	if iplist == nil || len(iplist) != len(table) {
		t.Fail()
	}
}

func TestNewRequestRegistry(t *testing.T) {
	r := NewRequestRegistry(10 * time.Second)
	if r == nil {
		t.Fail()
	}
}

func TestRegistRequest(t *testing.T) {
	registry := NewRequestRegistry(10 * time.Second)
	if registry == nil {
		t.Fail()
	}

	if !registry.Regist(NewEchoRequest(net.ParseIP("127.0.0.1"), 1)) {
		t.Fail()
	}
	if registry.Regist(NewEchoRequest(net.ParseIP("127.0.0.1"), 1)) {
		t.Fail()
	}
	if !registry.Regist(NewEchoRequest(net.ParseIP("127.0.0.1"), 2)) {
		t.Fail()
	}
}

func TestUnregistRequest(t *testing.T) {

}

func TestCleanupRequests(t *testing.T) {

}