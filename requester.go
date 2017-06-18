package nettools

import (
	"errors"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"log"
	"net"
	"time"
	"fmt"
)


func NewSocket() (*icmp.PacketConn, error) {
	c, err := icmp.ListenPacket("udp4", "")
	if err != nil {
		return nil, err
	}

	return c, nil
}

func NewRequester(dest net.IP, interval time.Duration, send chan *EchoMessage) {
	go func () {
		m := NewEchoMessage(dest, 0)
		for {
			select {
			case <-time.After(interval):
				send <- m
				m = m.Next()
			}
		}
	}()
}

func NewReceiver(c *icmp.PacketConn, received chan *EchoMessage) {
	go func() {
		rb := make([]byte, 1500)
		for {
			n, peer, err := c.ReadFrom(rb)
			if err != nil {
				log.Fatal(err)
			}

			peerIP := peer.String()
			peerIP = peerIP[:len(peerIP)-2]

			rm, err := icmp.ParseMessage(ProtocolICMP, rb[:n])
			if err != nil {
				log.Fatal(err)
			}

			if rm.Type == ipv4.ICMPTypeEchoReply {
				r := BuildEchoMessageFromResponse(rm)
				received<-r
			}
		}
	}()
}

func NewSender(c *icmp.PacketConn, send chan *EchoMessage, sent chan *EchoMessage) {
	go func() {
		for {
			select {
			case r := <-send:
				r.Timestamp = time.Now()
				wm := r.ICMPMessage()
				wb, err := wm.Marshal(nil)
				if err != nil {
					log.Fatal(err)
				}

				if _, err := c.WriteTo(wb, &net.UDPAddr{IP: r.IP}); err != nil {
					panic(err)
				}
				sent<-r
			}
		}
	}()
}

func NewReporter(sent chan *EchoMessage, received chan *EchoMessage, report chan *Report, timeout time.Duration) {
	list := make(map[string]*EchoMessage)

	timeoutCheckInterval := time.Second

	go func() {
		for {
			select {
			case req := <- sent:
				key := req.String()
				_, ok := list[key]
				if ok {
					report <- &Report {
						"Sent duplicated message",
						req,
					}
				} else {
					list[key] = req
				}
			case res := <- received:
				key := res.String()
				_, ok := list[key]
				if !ok {
					report <- &Report {
						"Unsent message get received",
						res,
					}
				} else {
					delete(list, key)
					report <- &Report {
						"Echo returned",
						res,
					}
				}
			case <-time.After(timeoutCheckInterval):
				// handle timeout
				expired := extractExpired(list, timeout)
				if len(expired) > 0 {
					fmt.Println(expired)
				}
			}
		}
	}()
}

type Report struct {
	Text string
	Message *EchoMessage
}

func NewPinger(timeout time.Duration) (chan *EchoMessage, chan *Report) {
	send := make(chan *EchoMessage, 100)
	sent := make(chan *EchoMessage, 100)
	received := make(chan *EchoMessage, 100)
	report := make(chan *Report, 100)


	c, err := NewSocket()
	if err != nil {
		panic(err)
	}

	NewReceiver(c, received)
	NewSender(c, send, sent)
	NewReporter(sent, received, report, timeout)

	return send, report
}

func extractExpired(list map[string]*EchoMessage, timeout time.Duration) []*EchoMessage {
	expiredkeys := make([]string, 0)

	for k, v := range list {
		now := time.Now()
		timeoutNano := timeout.Nanoseconds()
		if !v.Timestamp.IsZero() && now.Sub(v.Timestamp).Nanoseconds() >= timeoutNano {
			expiredkeys = append(expiredkeys, k)
		}
	}

	expired := make([]*EchoMessage, len(expiredkeys))
	for i := range expiredkeys {
		expired = append(expired, list[expiredkeys[i]])
		delete(list, expiredkeys[i])
	}

	return expired
}


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