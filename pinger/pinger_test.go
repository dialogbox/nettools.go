package pinger

import (
	"testing"
	"time"
)

func TestNewRequester(t *testing.T) {
	ip, _ := lookupIP("127.0.0.1")
	req := make(chan *EchoMessage, 1000)
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
	c := icmpsocket()
	defer c.Close()

	res := make(chan *EchoMessage, 1000)
	defer close(res)

	done := listener(c, res)
	done <- struct{}{}
}

func TestNewSender(t *testing.T) {
	c := icmpsocket()
	defer c.Close()

	req := make(chan *EchoMessage, 1000)
	defer close(req)

	sent := make(chan *EchoMessage, 1000)
	defer close(sent)

	ip, _ := lookupIP("127.0.0.1")
	done := sender(c, req, sent)
	req <- NewEchoMessage(ip, 0)

	s := <-sent
	done <- struct{}{}
	if !s.IP.Equal(ip) {
		t.Fail()
	}
}

func TestNewReporter(t *testing.T) {
	c := icmpsocket()
	defer c.Close()

	req := make(chan *EchoMessage, 1000)
	defer close(req)

	sent := make(chan *EchoMessage, 1000)
	defer close(sent)

	res := make(chan *EchoMessage, 1000)
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
			t.Logf("Report : %v\n", r)
		}
	}
}

func TestPinger(t *testing.T) {
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
			t.Logf("Report : %v\n", r)
		}
	}

}