package windowsaudio

import (
	"context"
	"testing"
	"time"
)

func TestRevocationWaitsAndSupersedesPolicy(t *testing.T) {
	requests := make(chan Request, 1)
	done := make(chan struct{})
	requests <- Request{Enabled: true, Live: true}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- Revoke(ctx, requests, done) }()
	var r Request
	deadline := time.After(time.Second)
	for r.Ack == nil {
		select {
		case r = <-requests:
		case <-deadline:
			t.Fatal("revocation not submitted")
		}
	}
	if r.Enabled || r.Live {
		t.Fatal("old permit retained")
	}
	select {
	case <-result:
		t.Fatal("acknowledged before old native work completed")
	default:
	}
	close(r.Ack)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	Update(requests, Request{Enabled: true})
	Update(requests, Request{})
	if (<-requests).Enabled {
		t.Fatal("new disable dropped")
	}
}
func TestRevocationTimeoutAndStoppedWorker(t *testing.T) {
	requests := make(chan Request, 1)
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if Revoke(ctx, requests, done) == nil {
		t.Fatal("unverified shutdown accepted")
	}
	close(done)
	if e := Revoke(context.Background(), requests, done); e != nil {
		t.Fatal(e)
	}
}
