//go:build windows

package streamdeck

import (
	"context"
	"testing"
	"time"
)

func TestAbsentDiscoveryIgnoresFrameBurst(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := make(chan time.Time, 8)
	frames, events, done := startSurface(ctx, func() ([]string, error) { calls <- time.Now(); return nil, nil })
	first := <-calls
	select {
	case event := <-events:
		if event.Error != "Stream Deck disconnected" {
			t.Fatal(event)
		}
	case <-time.After(time.Second):
		t.Fatal("missing disconnected state")
	}
	for n := 0; n < 100; n++ {
		select {
		case frames <- Frame{Generation: uint64(n)}:
		case <-time.After(time.Second):
			t.Fatal("blocked frame")
		}
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
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("leaked worker")
	}
}
