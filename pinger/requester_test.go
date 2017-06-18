package pinger

import (
	"testing"
	"time"
)

func TestNewRequester(t *testing.T) {
	ip, _ := LookupIPAddr("127.0.0.1")
	req := make(chan *EchoMessage, 1000)
	defer close(req)

	done := NewRequester(req, ip, 1*time.Second)
	time.Sleep(3 * time.Second)
	done <- struct{}{}

	r := <-req
	r = <-req

	if !r.IP.Equal(ip) {
		t.Fail()
	}
}

func TestNewListener(t *testing.T) {
	c := NewSocket()
	defer c.Close()

	res := make(chan *EchoMessage, 1000)
	defer close(res)

	done := NewListener(c, res)
	done <- struct{}{}
}

func TestNewSender(t *testing.T) {
	c := NewSocket()
	defer c.Close()

	req := make(chan *EchoMessage, 1000)
	defer close(req)

	sent := make(chan *EchoMessage, 1000)
	defer close(sent)

	ip, _ := LookupIPAddr("127.0.0.1")
	done := NewSender(c, req, sent)
	req <- NewEchoMessage(ip, 0)

	s := <-sent
	done <- struct{}{}
	if !s.IP.Equal(ip) {
		t.Fail()
	}
}

func TestNewReporter(t *testing.T) {
	c := NewSocket()
	defer c.Close()

	req := make(chan *EchoMessage, 1000)
	defer close(req)

	sent := make(chan *EchoMessage, 1000)
	defer close(sent)

	res := make(chan *EchoMessage, 1000)
	defer close(res)

	report := make(chan *Report, 1000)
	defer close(report)

	doneSender := NewSender(c, req, sent)
	doneListener := NewListener(c, res)

	ip, _ := LookupIPAddr("127.0.0.1")
	doneRequester := NewRequester(req, ip, 1*time.Second)

	done := NewReporter(report, sent, res, 5*time.Second)

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
			t.Logf("Report : %v\n", r)
		}
	}
}

func TestPinger(t *testing.T) {
	pinger := NewPinger(1000, 5*time.Second)

	ip, _ := LookupIPAddr("127.0.0.1")
	pinger.AddDest(ip, 2*time.Second)
	ip, _ = LookupIPAddr("127.0.0.2")
	pinger.AddDest(ip, 4*time.Second)

	timer := time.After(10 * time.Second)

	report := pinger.Start()

	for {
		select {
		case <-timer:
			pinger.Stop()
			return
		case r := <-report:
			t.Logf("Report : %v\n", r)
		}
	}

}
