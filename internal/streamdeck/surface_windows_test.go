//go:build windows

package streamdeck

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestAbsentDiscoveryIgnoresFrameBurst(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := make(chan time.Time, 8)
	surface := startSurface(ctx, func() ([]string, error) { calls <- time.Now(); return nil, nil }, nil)
	first := <-calls
	select {
	case event := <-surface.Events():
		if event.Error != "Stream Deck disconnected" || event.Device != "" {
			t.Fatal(event)
		}
	case <-time.After(time.Second):
		t.Fatal("missing disconnected state")
	}
	for n := 0; n < 100; n++ {
		surface.Show("A", Frame{Generation: uint64(n)})
	}
	select {
	case second := <-calls:
		if second.Sub(first) < 1900*time.Millisecond {
			t.Fatal("frame burst accelerated discovery")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("discovery stopped")
	}
	cancel()
	select {
	case <-surface.Done():
	case <-time.After(time.Second):
		t.Fatal("leaked worker")
	}
}

// fakeDeck is an open deck whose input and failures the test supplies.
type fakeDeck struct {
	info    deckInfo
	initial []byte
	input   chan []byte
	fail    chan error
	mu      sync.Mutex
	writes  int
	closed  bool
}

func newFakeDeck(serial, model string) *fakeDeck {
	m, _ := ModelByID(model)
	return &fakeDeck{info: deckInfo{input: 512, output: 1024, serial: serial, model: m}, input: make(chan []byte, 8), fail: make(chan error, 1)}
}

func (f *fakeDeck) io(ctx context.Context, b []byte, write bool) (int, error) {
	if write {
		f.mu.Lock()
		f.writes++
		f.mu.Unlock()
		return len(b), nil
	}
	select {
	case r := <-f.input:
		return copy(b, r), nil
	case err := <-f.fail:
		return 0, err
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-time.After(50 * time.Millisecond):
		return 0, context.DeadlineExceeded
	}
}
func (f *fakeDeck) inputReport() ([]byte, error) {
	if f.initial == nil {
		return nil, errors.New("current input unavailable")
	}
	return f.initial, nil
}
func (f *fakeDeck) setBrightness(int) error { return nil }
func (f *fakeDeck) describe() deckInfo      { return f.info }
func (f *fakeDeck) close() {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
}
func (f *fakeDeck) written() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.writes
}

// next waits for an event that satisfies want, skipping others.
func next(t *testing.T, events <-chan Event, want func(Event) bool) Event {
	t.Helper()
	deadline := time.After(4 * time.Second)
	for {
		select {
		case e, ok := <-events:
			if !ok {
				t.Fatal("events closed")
			}
			if want(e) {
				return e
			}
		case <-deadline:
			t.Fatal("event not seen")
		}
	}
}

func TestEachDeckRunsItsOwnSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	opened := map[string][]*fakeDeck{}
	open := func(path string) (hidDeck, error) {
		mu.Lock()
		defer mu.Unlock()
		d := map[string]*fakeDeck{"a": newFakeDeck("A", "plus"), "b": newFakeDeck("B", "plus-xl")}[path]
		opened[path] = append(opened[path], d)
		return d, nil
	}
	deck := func(path string) *fakeDeck {
		mu.Lock()
		defer mu.Unlock()
		list := opened[path]
		return list[len(list)-1]
	}
	surface := startSurface(ctx, func() ([]string, error) { return []string{"a", "b"}, nil }, open)
	events := surface.Events()
	connected := map[string]string{}
	for len(connected) < 2 {
		e := next(t, events, func(e Event) bool { return e.Connected })
		connected[e.Device] = e.Model
	}
	if connected["A"] != "plus" || connected["B"] != "plus-xl" {
		t.Fatal(connected)
	}

	// A frame for A is drawn on A only.
	a, b := deck("a"), deck("b")
	aBefore, bBefore := a.written(), b.written()
	frame := Frame{Generation: 7}
	frame.Keys[0] = Tile{Label: "Mic", Value: "On", Icon: "mic-mute"}
	surface.Show("A", frame)
	for deadline := time.Now().Add(2 * time.Second); a.written() == aBefore; {
		if time.Now().After(deadline) {
			t.Fatal("frame for A not drawn")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if b.written() != bBefore {
		t.Fatal("frame for A drawn on B")
	}

	// Input carries its deck.
	b.input <- report(0, make([]byte, 36))
	keys := make([]byte, 36)
	keys[3] = 1
	b.input <- report(0, keys)
	if e := next(t, events, func(e Event) bool { return e.Press }); e.Device != "B" || e.Key != 3 || e.Model != "plus-xl" {
		t.Fatal(e)
	}

	// A failing leaves B running, and A reconnects with its last frame.
	a.fail <- errors.New("unplugged")
	if e := next(t, events, func(e Event) bool { return e.Error != "" }); e.Device != "A" {
		t.Fatal(e)
	}
	b.input <- report(0, make([]byte, 36))
	b.input <- report(0, keys)
	if e := next(t, events, func(e Event) bool { return e.Press }); e.Device != "B" {
		t.Fatal("B stopped with A", e)
	}
	if e := next(t, events, func(e Event) bool { return e.Connected }); e.Device != "A" {
		t.Fatal(e)
	}
	if again := deck("a"); again == a || again.written() == 0 {
		t.Fatal("A not reopened and redrawn")
	}

	cancel()
	select {
	case <-surface.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("sessions did not stop")
	}
	mu.Lock()
	defer mu.Unlock()
	for path, list := range opened {
		for _, d := range list {
			if !d.closed {
				t.Fatal(path, "left open")
			}
		}
	}
}
