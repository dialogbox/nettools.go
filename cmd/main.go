package main

import (
	"fmt"
	"sync"

	"github.com/dialogbox/nettools"
	"golang.org/x/net/icmp"
)


func main() {

	t := []string{
		"127.0.0.1",
		"www.google.com",
		"www.apple.com",
	}

	p, err := nettools.NewPinger()
	if err != nil {
		panic(err)
	}
	defer p.Close()

	for i := range t {
		go p.Ping(t[i])
	}

	var wg sync.WaitGroup
	wg.Add(len(t))

	for range t {
		go func() {
			defer wg.Done()
			peer, rm, err := p.GetPacket()
			if err != nil {
				panic(err)
			}
			e := rm.Body.(*icmp.Echo)
			fmt.Printf("%v, %v, %v, %v\n", *peer, e.ID, e.Seq, e.Data)
		}()
	}

	wg.Wait()

}
