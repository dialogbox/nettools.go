package nettools

import (
	"errors"
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"net"
	"time"
)

func NewSocket() *icmp.PacketConn {
	c, err := icmp.ListenPacket("udp4", "")
	if err != nil {
		panic(err)
	}

	return c
}

func NewRequester(send chan *EchoMessage, dest net.IP, interval time.Duration) chan struct{} {
	done := make(chan struct{})
	go func() {
		m := NewEchoMessage(dest, 0)
		for {
			select {
			case <-done:
				close(done)
				return
			case <-time.After(interval):
				send <- m
				m = m.Next()
			}
		}
	}()
	return done
}

func receiveOneMessage(c *icmp.PacketConn, buf []byte) *icmp.Message {
	c.SetReadDeadline(time.Now().Add(time.Second))
	n, _, err := c.ReadFrom(buf)
	if err != nil {
		switch err := err.(type) {
		case net.Error:
			if err.Timeout() {
				return nil
			} else {
				panic(err)
			}
		default:
			panic(err)
		}
	}

	rm, err := icmp.ParseMessage(ProtocolICMP, buf[:n])
	if err != nil {
		panic(err)
	}

	return rm
}

func NewListener(c *icmp.PacketConn, resBufSize int) (chan *EchoMessage, chan struct{}) {
	done := make(chan struct{})
	res := make(chan *EchoMessage, resBufSize)

	go func() {
		rb := make([]byte, 1500)
		for {
			select {
			case <-done:
				close(done)
				close(res)
				return
			default:
				rm := receiveOneMessage(c, rb)
				if rm != nil && rm.Type == ipv4.ICMPTypeEchoReply {
					r := BuildEchoMessageFromResponse(rm)
					res <- r
				}
			}
		}
	}()

	return res, done
}

func NewSender(c *icmp.PacketConn) (chan *EchoMessage, chan *EchoMessage, chan struct{}) {
	done := make(chan struct{})
	req := make(chan *EchoMessage, 1000)
	sent := make(chan *EchoMessage, 1000)

	go func() {
		for {
			select {
			case <-done:
				close(done)
				close(req)
				close(sent)
				return
			case r := <-req:
				r.Timestamp = time.Now()
				wm := r.ICMPMessage()
				wb, err := wm.Marshal(nil)
				if err != nil {
					panic(err)
				}

				if _, err := c.WriteTo(wb, &net.UDPAddr{IP: r.IP}); err != nil {
					panic(err)
				}
				sent <- r
			}
		}
	}()

	return req, sent, done
}

func NewReporter(sent chan *EchoMessage, res chan *EchoMessage, timeout time.Duration) (chan *Report, chan struct{}) {
	done := make(chan struct{})
	report := make(chan *Report, 1000)

	list := make(map[string]*EchoMessage)

	timeoutCheckInterval := time.Second

	go func() {
		for {
			select {
			case <-done:
				close(report)
				close(done)

				return
			case r := <-sent:
				key := r.String()
				_, ok := list[key]
				if ok {
					report <- &Report{
						"Sent duplicated message",
						r,
					}
				} else {
					list[key] = r
				}
			case r := <-res:
				key := r.String()
				_, ok := list[key]
				if !ok {
					report <- &Report{
						"Unsent message get received",
						r,
					}
				} else {
					delete(list, key)
					report <- &Report{
						"Echo returned",
						r,
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

	return report, done
}

type Report struct {
	Text    string
	Message *EchoMessage
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
