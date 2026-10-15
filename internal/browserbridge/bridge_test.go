package browserbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func newTestBridge(t *testing.T) *Bridge {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	b := New(context.Background(), json.RawMessage(fmt.Sprintf(`{"port":%d}`, port)))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := b.Close(ctx); err != nil {
			t.Error("bridge cleanup:", err)
		}
	})
	return b
}

func mediaLease(t *testing.T, b *Bridge) *MediaLease {
	t.Helper()
	l, err := b.AcquireMedia(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func meetLease(t *testing.T, b *Bridge) *MeetLease {
	t.Helper()
	l, err := b.AcquireMeet(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func connect(t *testing.T, b *Bridge, browser string, meet int) *websocket.Conn {
	t.Helper()
	ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://127.0.0.1:%d/nowplaying", b.Status().Port), http.Header{"Origin": []string{"moz-extension://test"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ws.Close() })
	writeJSON(t, ws, hello{Type: "hello", Protocol: protocol, Browser: browser, Version: "test", Meet: meet})
	return ws
}

func writeJSON(t *testing.T, ws *websocket.Conn, value any) {
	t.Helper()
	if err := ws.WriteJSON(value); err != nil {
		t.Fatal(err)
	}
}

func readJSON(t *testing.T, ws *websocket.Conn, value any) {
	t.Helper()
	if err := ws.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := ws.ReadJSON(value); err != nil {
		t.Fatal(err)
	}
}

func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v, ok := <-ch:
		if !ok {
			t.Fatal("channel closed early")
		}
		return v
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for update")
		var zero T
		return zero
	}
}

func eventually(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !ready() {
		if time.Now().After(deadline) {
			t.Fatal("condition did not become true")
		}
		time.Sleep(time.Millisecond)
	}
}

func closeLease(t *testing.T, l *lease) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := l.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestLeasesShareSocketAndStopIndependently(t *testing.T) {
	for _, mediaFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(mediaFirst), func(t *testing.T) {
			b := newTestBridge(t)
			var media *MediaLease
			var meet *MeetLease
			if mediaFirst {
				media = mediaLease(t, b)
			} else {
				meet = meetLease(t, b)
			}
			ws := connect(t, b, "Browser", 1)
			var caps capabilities
			readJSON(t, ws, &caps)
			if caps.MediaEnabled != mediaFirst || caps.MeetEnabled == mediaFirst {
				t.Fatal(caps)
			}
			if mediaFirst {
				receive(t, media.Updates())
				meet = meetLease(t, b)
			} else {
				receive(t, meet.Updates())
				media = mediaLease(t, b)
			}
			readJSON(t, ws, &caps)
			if !caps.MediaEnabled || !caps.MeetEnabled {
				t.Fatal(caps)
			}
			generation := meet.Generation("Browser")
			if generation == 0 || media.Generation("Browser") != generation {
				t.Fatal("separate connections")
			}
			if _, err := b.AcquireMedia(context.Background()); err == nil {
				t.Fatal("duplicate media lease")
			}
			if _, err := b.AcquireMeet(context.Background()); err == nil {
				t.Fatal("duplicate meet lease")
			}
			if mediaFirst {
				closeLease(t, media.lease)
				if meet.Generation("Browser") != generation {
					t.Fatal("Media close lost Meet")
				}
			} else {
				closeLease(t, meet.lease)
				if media.Generation("Browser") != generation {
					t.Fatal("Meet close lost Media")
				}
			}
			readJSON(t, ws, &caps)
			if caps.MediaEnabled == mediaFirst || caps.MeetEnabled != mediaFirst {
				t.Fatal(caps)
			}
			if mediaFirst {
				closeLease(t, meet.lease)
			} else {
				closeLease(t, media.lease)
			}
			if _, _, err := ws.ReadMessage(); err == nil {
				t.Fatal("last close retained socket")
			}
			again := mediaLease(t, b)
			ws2 := connect(t, b, "Browser", 1)
			readJSON(t, ws2, &caps)
			if current := receive(t, again.Updates()); current.Generation <= generation {
				t.Fatal("generation reused", current)
			}
		})
	}
}

func TestLateMediaRefreshesOldExtensionWithoutMeetSupport(t *testing.T) {
	b := newTestBridge(t)
	meet := meetLease(t, b)
	ws := connect(t, b, "Old", 0)
	var caps capabilities
	readJSON(t, ws, &caps)
	u := receive(t, meet.Updates())
	if u.Supported || caps.MeetEnabled {
		t.Fatal("old extension claimed support")
	}
	media := mediaLease(t, b)
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("old connection not refreshed")
	}
	if media.Generation("Old") != 0 {
		t.Fatal("old generation retained")
	}
	ws2 := connect(t, b, "Old", 0)
	readJSON(t, ws2, &caps)
	if !caps.MediaEnabled || caps.MeetEnabled {
		t.Fatal(caps)
	}
}

func TestMediaFIFOIsOrderedAndOverflowRetiresOnlyItsSocket(t *testing.T) {
	for _, feature := range []string{"media", "meet"} {
		t.Run(feature, func(t *testing.T) {
			b := newTestBridge(t)
			media := mediaLease(t, b)
			meet := meetLease(t, b)
			a := connect(t, b, "A", 1)
			other := connect(t, b, "B", 1)
			var caps capabilities
			readJSON(t, a, &caps)
			readJSON(t, other, &caps)
			receive(t, media.Updates())
			receive(t, media.Updates())
			receive(t, meet.Updates())
			receive(t, meet.Updates())
			old := media.Generation("A")
			good := media.Generation("B")
			b.mu.Lock()
			c := b.conns["A"]
			for n := 0; n < 16; n++ {
				if feature == "media" {
					b.mediaLocked(c, []MediaSession{{ID: "7:0", Instance: "1", ArtKey: fmt.Sprint(n), Art: fmt.Sprint(n)}})
				} else {
					b.meetLocked(c, nil)
				}
			}
			if feature == "media" {
				b.mediaLocked(c, nil)
			} else {
				b.meetLocked(c, nil)
			}
			b.mu.Unlock()
			if old == 0 || media.Generation("A") != 0 || meet.Generation("A") != 0 || media.Generation("B") != good {
				t.Fatal("wrong retirement")
			}
			if feature == "media" {
				for n := 0; n < 16; n++ {
					u := receive(t, media.Updates())
					if u.Generation != old || u.Sessions[0].Art != fmt.Sprint(n) {
						t.Fatal("one-shot artwork reordered", u)
					}
				}
			}
			if err := media.Send("A", MediaCommand{Generation: old, Type: "command"}); !errors.Is(err, ErrConnectionChanged) {
				t.Fatal(err)
			}
		})
	}
}

func TestShutdownClosesPartialHelloAndReleasesPort(t *testing.T) {
	b := newTestBridge(t)
	media := mediaLease(t, b)
	address := fmt.Sprintf("127.0.0.1:%d", b.Status().Port)
	ws, _, err := websocket.DefaultDialer.Dial("ws://"+address+"/nowplaying", http.Header{"Origin": []string{"moz-extension://test"}})
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	closeLease(t, media.lease)
	if err := ws.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("hello socket not closed")
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatal("listener remains owned", err)
	}
	listener.Close()
}

func TestClosingLeaseWhileConsumerDrainsDoesNotBlock(t *testing.T) {
	// Exercise the exact queue-drain boundary with a competing receiver. Empty
	// buffers and the last item are both normal, not grounds for a blocking read.
	for range 1000 {
		updates := make(chan MediaUpdate, 16)
		for range cap(updates) {
			updates <- MediaUpdate{}
		}
		var consumer sync.WaitGroup
		consumer.Add(1)
		go func() {
			defer consumer.Done()
			for range updates {
			}
		}()
		done := make(chan struct{})
		go func() { closeUpdates(updates); close(done) }()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("queue close blocked")
		}
		consumer.Wait()
	}
}

func TestWriteWaitingAtGateRejectsOwnerShutdown(t *testing.T) {
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
	finished := make(chan error, 1)
	go func() {
		finished <- media.Send("Browser", MediaCommand{Type: "command", Generation: u.Generation, Op: "next"})
	}()
	// This is the first, mutex-protected part of owner Close, before individual
	// lease cancellation can run. Releasing the write gate must still reject.
	b.mu.Lock()
	b.closed = true
	c.run.stopping = true
	c.run.cancel()
	b.mu.Unlock()
	if media.Generation("Browser") != 0 {
		t.Fatal("shutdown retained generation")
	}
	<-c.gate
	if err := receive(t, finished); !errors.Is(err, ErrConnectionChanged) {
		t.Fatal("shutdown allowed write", err)
	}
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("command reached stopped connection")
	}
}

func TestBusyPortRetryStopsWithFinalLease(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	b := New(context.Background(), json.RawMessage(fmt.Sprintf(`{"port":%d}`, busy.Addr().(*net.TCPAddr).Port)))
	b.retry = time.Millisecond
	t.Cleanup(func() { busy.Close(); b.Close(context.Background()) })
	media := mediaLease(t, b)
	if b.Status().Error == "" {
		t.Fatal("busy port hidden")
	}
	busy.Close()
	eventually(t, func() bool { return b.Status().Error == "" })
	closeLease(t, media.lease)
	if b.run != nil {
		t.Fatal("retry remains owned")
	}
}
