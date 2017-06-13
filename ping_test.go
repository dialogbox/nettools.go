package nettools

import (
	"golang.org/x/net/icmp"
	"net"
	"testing"
)
//
//func TestNewPinger(t *testing.T) {
//	p, err := NewPinger()
//	if err != nil {
//		panic(err)
//	}
//	defer p.Close()
//}
//
//func TestPing(t *testing.T) {
//	table := []string{
//		"127.0.0.1",
//		"www.google.com",
//		"www.apple.com",
//	}
//
//	p, err := NewPinger()
//	if err != nil {
//		panic(err)
//	}
//	defer p.Close()
//
//	for i := range table {
//		go p.Ping(table[i])
//	}
//
//	var wg sync.WaitGroup
//	wg.Add(len(table))
//
//	for range table {
//		go func() {
//			defer wg.Done()
//			peer, rm, err := p.GetPacket()
//			if err != nil {
//				panic(err)
//			}
//			e := rm.Body.(*icmp.Echo)
//			fmt.Printf("%v, %v, %v, %v\n", *peer, e.ID, e.Seq, e.Data)
//		}()
//	}
//
//	wg.Wait()
//}

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

func TestMakeHostDirectory(t *testing.T) {
	table := []string{
		"127.0.0.1",
		"www.google.com",
		"www.apple.com",
	}

	iplist := ResolveHostList(table)
	dir := MakeHostDirectory(iplist)
	if dir == nil || len(dir) != len(table) {
		t.Fail()
	}
}

func MakeHostDirectory(iplist []net.IP) []*icmp.Echo {
	dir := make([]*icmp.Echo, len(iplist))
	for i := range iplist {
		dir[i] = NewEchoRequest(iplist[i])
	}
	return dir
}

type Host struct {

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