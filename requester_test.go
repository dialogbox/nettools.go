package nettools

import (
	"testing"
	"time"
	"fmt"
)

func TestNewPingRequester(t *testing.T) {
	send, report := NewPinger(5 * time.Second)

	ip, _ := LookupIPAddr("127.0.0.1")
	NewRequester(ip, time.Second, send)

	ip2, _ := LookupIPAddr("127.0.0.2")
	NewRequester(ip2, time.Second, send)

	for {
		select {
		case r := <-report:
			fmt.Printf("R: %v\n", r)
		}
	}
}