package pinger

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const ProtocolICMP = 1
const ProtocolIPv6ICMP = 58

type Pinger struct {
	c *icmp.PacketConn

	req    chan *EchoRequest
	res    chan *EchoReply
	sent   chan *EchoRequest
	report chan *Report

	stopSender   chan struct{}
	stopReceiver chan struct{}
	stopReporter chan struct{}

	requesters map[string]chan struct{}

	id        uint16
	timeout   time.Duration
	queueSize int

	started bool
}

func NewPinger(queueSize int, timeout time.Duration) *Pinger {
	p := &Pinger{
		requesters: make(map[string]chan struct{}),
		timeout:    timeout,
		queueSize:  queueSize,
		req:        make(chan *EchoRequest, queueSize),
		res:        make(chan *EchoReply, queueSize),
		sent:       make(chan *EchoRequest, queueSize),
		report:     make(chan *Report, queueSize),
	}

	return p
}

func (p *Pinger) Start() chan *Report {
	p.c = icmpsocket()
	p.stopSender = sender(p.c, p.req, p.sent)
	p.stopReceiver = listener(p.c, p.res)
	p.stopReporter = reporter(p.report, p.sent, p.res, p.timeout)

	p.started = true

	return p.report
}

func (p *Pinger) Stop() {
	p.started = false

	for _, v := range p.requesters {
		v <- struct{}{}
	}

	p.stopReceiver <- struct{}{}
	p.stopReporter <- struct{}{}
	p.stopSender <- struct{}{}

	p.c.Close()

	close(p.req)
	close(p.res)
	close(p.report)
	close(p.sent)
}

func (p *Pinger) AddDest(ip net.IP, interval time.Duration) error {
	addr := ip.String()
	_, ok := p.requesters[addr]
	if ok {
		return fmt.Errorf("Duplicated address: %v", addr)
	}

	done := requester(p.req, ip, 1*time.Second)
	p.requesters[addr] = done

	return nil
}

func (p *Pinger) DeleteDest(ip net.IP) error {
	addr := ip.String()
	_, ok := p.requesters[addr]
	if !ok {
		return fmt.Errorf("Not registered address: %v", addr)
	}

	deleted := p.requesters[addr]
	delete(p.requesters, addr)

	deleted <- struct{}{}

	return nil
}

func icmpsocket() *icmp.PacketConn {
	c, err := icmp.ListenPacket("udp4", "")
	if err != nil {
		panic(err)
	}

	return c
}

func requester(send chan *EchoRequest, dest net.IP, interval time.Duration) chan struct{} {
	done := make(chan struct{})
	go func() {
		m := NewEchoRequest(dest, 0)
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

func listener(c *icmp.PacketConn, res chan *EchoReply) chan struct{} {
	done := make(chan struct{})

	go func() {
		rb := make([]byte, 1500)
		for {
			select {
			case <-done:
				close(done)
				return
			default:
				rm := receiveOneMessage(c, rb)
				if rm != nil && rm.Type == ipv4.ICMPTypeEchoReply {
					r := NewEchoReplyFromResponse(rm)
					if r != nil {
						res <- r
					}
				}
			}
		}
	}()

	return done
}

func sender(c *icmp.PacketConn, req chan *EchoRequest, sent chan *EchoRequest) chan struct{} {
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				close(done)
				return
			case r := <-req:
				r.RequestTime = time.Now()
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

	return done
}

func reporter(report chan *Report, sent chan *EchoRequest, res chan *EchoReply, timeout time.Duration) chan struct{} {
	done := make(chan struct{})

	sentMsgs := make(map[string]*EchoRequest)

	timeoutCheckInterval := time.Second

	go func() {
		for {
			select {
			case <-done:
				close(done)

				return
			case r := <-sent:
				key := r.HashKey()
				_, ok := sentMsgs[key]
				if ok {
					report <- &Report{
						REPORT_EVENT_DUPLICATED_REQUEST,
						r,
						nil,
					}
				} else {
					sentMsgs[key] = r
				}
			case r := <-res:
				key := r.HashKey()
				old, ok := sentMsgs[key]
				if !ok {
					report <- &Report{
						REPORT_EVENT_UNKNOWN_REPLY,
						nil,
						r,
					}
				} else {
					delete(sentMsgs, key)
					report <- &Report{
						REPORT_EVENT_ECHO_RECEIVED,
						old,
						r,
					}
				}
			case <-time.After(timeoutCheckInterval):
				// handle timeout
				expired := extractExpired(sentMsgs, timeout)
				if len(expired) > 0 {
					for i := range expired {
						report <- &Report{
							REPORT_EVENT_PACKET_LOSS,
							expired[i],
							nil,
						}
					}
				}
			}
		}
	}()

	return done
}

func extractExpired(list map[string]*EchoRequest, timeout time.Duration) []*EchoRequest {
	var expiredkeys []string

	now := time.Now()
	timeoutNano := timeout.Nanoseconds()
	for k, v := range list {
		if now.Sub(v.RequestTime).Nanoseconds() >= timeoutNano {
			expiredkeys = append(expiredkeys, k)
		}
	}

	var expired []*EchoRequest
	for i := range expiredkeys {
		expired = append(expired, list[expiredkeys[i]])
		delete(list, expiredkeys[i])
	}

	return expired
}
