package pinger

import (
	"fmt"
	"testing"
	"time"
)

func TestNewRequester(t *testing.T) {
	ip, _ := lookupIP("127.0.0.1")
	req := make(chan *EchoRequest, 1000)
	defer close(req)

	done := requester(req, ip, 1*time.Second)
	time.Sleep(3 * time.Second)
	done <- struct{}{}

	r := <-req
	r = <-req

	if !r.IP.Equal(ip) {
		t.Fail()
	}
}

func TestNewListener(t *testing.T) {
	c := icmpSocket()
	defer c.Close()

	res := make(chan *EchoReply, 1000)
	defer close(res)

	done := listener(c, res)
	done <- struct{}{}
}

func TestNewSender(t *testing.T) {
	c := icmpSocket()
	defer c.Close()

	req := make(chan *EchoRequest, 1000)
	defer close(req)

	sent := make(chan *EchoRequest, 1000)
	defer close(sent)

	ip, _ := lookupIP("127.0.0.1")
	done := sender(c, req, sent)
	req <- NewEchoRequest(ip, 0)

	s := <-sent
	done <- struct{}{}
	if !s.IP.Equal(ip) {
		t.Fail()
	}
}

func TestNewReporter(t *testing.T) {
	c := icmpSocket()
	defer c.Close()

	req := make(chan *EchoRequest, 1000)
	defer close(req)

	sent := make(chan *EchoRequest, 1000)
	defer close(sent)

	res := make(chan *EchoReply, 1000)
	defer close(res)

	report := make(chan *Report, 1000)
	defer close(report)

	doneSender := sender(c, req, sent)
	doneListener := listener(c, res)

	ip, _ := lookupIP("127.0.0.1")
	doneRequester := requester(req, ip, 1*time.Second)

	done := reporter(report, sent, res, 5*time.Second)

	timer := time.After(10 * time.Second)

	for {
		select {
		case <-timer:
			doneRequester <- struct{}{}
			done <- struct{}{}
			doneSender <- struct{}{}
			doneListener <- struct{}{}

			return
		case r := <-report:
			fmt.Printf("Report : %v\n", r)
		}
	}
}

func TestPinger_AddDest(t *testing.T) {
	pinger := NewPinger(1000, 5*time.Second)

	ip, _ := lookupIP("127.0.0.1")
	err := pinger.AddDest(ip, 2*time.Second)
	if err != nil {
		t.Fail()
	}

	ip, _ = lookupIP("127.0.0.1")
	err = pinger.AddDest(ip, 2*time.Second)
	if err == nil {
		t.Fail() // Should be failed
	}
}

func TestPinger_Start(t *testing.T) {
	pinger := NewPinger(1000, 5*time.Second)

	ip, _ := lookupIP("127.0.0.1")
	pinger.AddDest(ip, 2*time.Second)

	ip, _ = lookupIP("127.0.0.2")
	pinger.AddDest(ip, 4*time.Second)

	timer := time.After(10 * time.Second)

	report := pinger.Start()

	for {
		select {
		case <-timer:
			pinger.Stop()
			return
		case r := <-report:
			fmt.Println(r)
		}
	}
}

func TestPinger_Start2(t *testing.T) {
	pinger := NewPinger(1000, 5*time.Second)

	ip, _ := lookupIP("127.0.0.1")
	pinger.AddDest(ip, 1*time.Second)

	ip, _ = lookupIP("8.8.8.8")
	pinger.AddDest(ip, 2*time.Second)

	ip, _ = lookupIP("8.8.4.4")
	pinger.AddDest(ip, 3*time.Second)

	ip, _ = lookupIP("www.naver.com")
	pinger.AddDest(ip, 5*time.Second)

	timer := time.After(100 * time.Second) // Long term test

	report := pinger.Start()

	for {
		select {
		case <-timer:
			pinger.Stop()
			return
		case r := <-report:
			fmt.Println(r)
		}
	}
}

