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

	_, done := NewListener(c, 1000)
	done <- struct{}{}
}

func TestNewSender(t *testing.T) {
	c := NewSocket()
	defer c.Close()

	ip, _ := LookupIPAddr("127.0.0.1")
	req, sent, done := NewSender(c)
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

	req, sent, doneSender := NewSender(c)
	res, doneListener := NewListener(c, 1000)

	ip, _ := LookupIPAddr("127.0.0.1")
	doneRequester := NewRequester(req, ip, 1*time.Second)

	report, done := NewReporter(sent, res, 5 * time.Second)

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