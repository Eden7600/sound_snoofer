//go:build windows

package streamdeck

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestSessionInitialKeyBaseline(t *testing.T) {
	up := report(0, make([]byte, 8))
	keys := make([]byte, 8)
	keys[0] = 1
	down := report(0, keys)
	press := Event{Key: 0, Encoder: -1, Press: true}
	release := Event{Key: 0, Encoder: -1, Release: true}
	for _, tt := range []struct {
		name    string
		initial []byte
		reports [][]byte
		want    []Event
	}{
		{"first press", up, [][]byte{down}, []Event{press}},
		{"duplicate initial report", down, [][]byte{down, up, down}, []Event{release, press}},
		{"refused baseline", nil, [][]byte{down, up}, []Event{release}},
		{"invalid baseline", []byte{1, 0, 0, 0}, [][]byte{down, up}, []Event{release}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d := newFakeDeck("A", "plus")
			d.initial = tt.initial
			events := make(chan Event, 16)
			done := make(chan struct{})
			go func() {
				defer close(done)
				surfaceSession(ctx, d, "A", Frame{Generation: 9}, make(chan Frame), events)
			}()
			defer func() {
				cancel()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Error("session did not stop")
				}
			}()
			next(t, events, func(e Event) bool { return e.Connected })
			for _, r := range tt.reports {
				d.input <- r
			}
			// A dial turn marks the end of the reports without a timing-based assertion.
			d.input <- report(3, []byte{1, 1, 0, 0, 0})
			var got []Event
			for {
				e := next(t, events, func(Event) bool { return true })
				if e.Device != "A" || e.Generation != 9 {
					t.Fatal("input lost its device or frame generation", e)
				}
				if e.Delta != 0 {
					break
				}
				got = append(got, Event{Key: e.Key, Encoder: e.Encoder, Press: e.Press, Release: e.Release})
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("key edges = %+v, want %+v", got, tt.want)
			}
		})
	}
}
