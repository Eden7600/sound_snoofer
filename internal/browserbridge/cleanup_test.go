package browserbridge

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestMediaGateTimeoutIsKnownUnsent(t *testing.T) {
	b := newTestBridge(t)
	media := mediaLease(t, b)
	ws := connect(t, b, "Browser", 1)
	var caps capabilities
	readJSON(t, ws, &caps)
	u := receive(t, media.Updates())
	b.mu.Lock()
	c := b.conns["Browser"]
	b.mu.Unlock()
	c.gate <- struct{}{}
	started := time.Now()
	err := media.Send("Browser", MediaCommand{Type: "command", Generation: u.Generation, Op: "next"})
	<-c.gate
	if !errors.Is(err, ErrBusy) || time.Since(started) > 3*time.Second {
		t.Fatalf("gate expiry was not bounded/known unsent: %v after %v", err, time.Since(started))
	}
	if err := media.Send("Browser", MediaCommand{Type: "command", Generation: u.Generation, Op: "prev"}); err != nil {
		t.Fatal(err)
	}
	var only MediaCommand
	readJSON(t, ws, &only)
	if only.Op != "prev" {
		t.Fatal("expired operation was written", only)
	}
}

func TestPartialHTTPShutdownFailureIsReportedAndPortReleased(t *testing.T) {
	// A connection still waiting for HTTP headers is owned by net/http rather
	// than the hijacked-socket map. Graceful timeout must close it and report it.
	b := newTestBridge(t)
	accepted := make(chan struct{}, 1)
	address := fmt.Sprintf("127.0.0.1:%d", b.Status().Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, stop := context.WithCancel(b.ctx)
	r := &listenerRun{ctx: runCtx, cancel: stop, done: make(chan struct{}), sockets: map[*websocket.Conn]bool{}}
	// Install the observation callback before Serve; no server fields change
	// concurrently with net/http. Cleanup itself uses the production owner.
	r.server = &http.Server{ReadHeaderTimeout: helloTimeout, ConnState: func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			accepted <- struct{}{}
		}
	}}
	b.run = r
	r.wg.Add(1)
	go func() { defer r.wg.Done(); _ = r.server.Serve(listener) }()
	go b.runListener(r)
	media := mediaLease(t, b)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	receive(t, accepted)
	if _, err := conn.Write([]byte("GET /nowplaying HTTP/1.1\r\nHost: localhost\r\n")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := media.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("graceful shutdown failure hidden", err)
	}
	if err := media.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("idempotent close lost failure", err)
	}
	if err := b.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("owner cleanup lost failure", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Fatal("partial HTTP socket still open")
	}
	again, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatal("listener remains owned", err)
	}
	again.Close()
	// The error was asserted above; do not report it again from the generic rig.
	b.mu.Lock()
	b.cleanupErr = nil
	b.mu.Unlock()
}
