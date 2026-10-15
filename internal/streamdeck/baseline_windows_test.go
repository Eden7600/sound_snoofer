//go:build windows

package streamdeck

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestSessionFirstButtonReports(t *testing.T) {
	for _, modelID := range []string{"plus", "plus-xl"} {
		m, _ := ModelByID(modelID)
		for _, kind := range []string{"key", "dial"} {
			for _, prefix := range []string{"clean", "invalid reports"} {
				t.Run(modelID+"/"+kind+"/"+prefix, func(t *testing.T) {
					var up, down []byte
					var press Event
					if kind == "key" {
						keys := make([]byte, m.Keys())
						up = report(0, keys)
						keys[0] = 1
						down = report(0, keys)
						press = Event{Key: 0, Encoder: -1, Press: true}
					} else {
						buttons := make([]byte, m.Dials+1)
						up = report(3, buttons)
						buttons[1] = 1
						down = report(3, buttons)
						press = Event{Key: -1, Encoder: 0, Press: true}
					}
					want := []Event{press}
					if kind == "key" {
						want = append(want, Event{Key: press.Key, Encoder: -1, Release: true})
					}
					want = append(want, press)

					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					d := newFakeDeck("A", modelID)
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
					if prefix == "invalid reports" {
						// The observed unsupported query reply and truncated key/dial reports.
						for _, bad := range [][]byte{{1, 0, 0, 0}, {1, 0, byte(m.Keys()), 0, 1}, report(3, []byte{0})} {
							d.input <- bad
						}
						// A later invalid state must not consume the earlier valid down.
						keys, buttons := make([]byte, m.Keys()), make([]byte, m.Dials+1)
						keys[0], keys[len(keys)-1] = 1, 2
						buttons[1], buttons[len(buttons)-1] = 1, 2
						d.input <- report(0, keys)
						d.input <- report(3, buttons)
					}
					for _, r := range [][]byte{down, down, up, down} {
						d.input <- r
					}
					// A dial turn marks the end without a timing-based negative assertion.
					turn := make([]byte, m.Dials+1)
					turn[0], turn[1] = 1, 1
					d.input <- report(3, turn)
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
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("button edges = %+v, want %+v", got, want)
					}
				})
			}
		}
	}
}
