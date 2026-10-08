//go:build windows

package streamdeck

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"sound-snoofer/internal/presence"
	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

// While locked or with the monitors off the deck shows a dark frame and
// dispatches nothing; afterwards it returns at the configured brightness.
func TestDarkDeckIsInert(t *testing.T) {
	presences := make(chan presence.State, 1)
	previous := watchPresence
	watchPresence = func(ctx context.Context) (<-chan presence.State, <-chan struct{}) {
		done := make(chan struct{})
		go func() { <-ctx.Done(); close(done) }()
		return presences, done
	}
	defer func() { watchPresence = previous }()

	ctx, cancel := context.WithCancel(context.Background())
	registry := snoofer.NewControls()
	var presses atomic.Int32
	registry.Publish("test", []snoofer.Control{{ID: "test.button", Label: "Button", Kind: "command", Operations: []string{"press"}, Available: true}}, func(context.Context, snoofer.Request) error {
		presses.Add(1)
		return nil
	})
	frames := make(chan device.Frame, 1)
	events := make(chan device.Event, 16)
	surfaceDone := make(chan struct{})
	surface := func(ctx context.Context) (chan device.Frame, <-chan device.Event, <-chan struct{}) {
		go func() { <-ctx.Done(); close(surfaceDone) }()
		return frames, events, surfaceDone
	}
	page := Page{ID: "a", Name: "A"}
	page.Keys[0] = Binding{Control: "test.button", Label: "test.button"}
	brightness := 70
	instance, err := startWithSurface(ctx, snoofer.Services{Controls: registry}, snoofer.MarshalSettings(Settings{Layout: Layout{Home: "a", Pages: []Page{page}}, Brightness: &brightness}), surface)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		stop, release := context.WithTimeout(context.Background(), time.Second)
		defer release()
		if err := instance.Stop(stop); err != nil {
			t.Error(err)
		}
	}()
	// frame returns the latest frame once frames settle and it satisfies ok;
	// presses must carry the generation the deck is showing.
	frame := func(ok func(device.Frame) bool) device.Frame {
		t.Helper()
		deadline := time.After(3 * time.Second)
		var latest device.Frame
		seen := false
		for {
			select {
			case latest = <-frames:
				seen = true
			case <-time.After(150 * time.Millisecond):
				if seen && ok(latest) {
					return latest
				}
			case <-deadline:
				t.Fatal("frame not reached", latest)
			}
		}
	}
	generation := uint64(0)
	press := func() {
		events <- device.Event{Key: 0, Encoder: -1, Press: true, Generation: generation}
		events <- device.Event{Key: 0, Encoder: -1, Release: true, Generation: generation}
	}
	events <- device.Event{Connected: true, Serial: "A1"}
	lit := frame(func(f device.Frame) bool { return !f.Dark && f.Keys[0].Label != "" })
	if lit.Brightness != 70 {
		t.Fatal("brightness", lit.Brightness)
	}
	// Lit, a press dispatches.
	generation = lit.Generation
	press()
	for deadline := time.Now().Add(2 * time.Second); presses.Load() != 1; time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("a lit deck did not dispatch")
		}
	}
	presences <- presence.State{Known: true, Locked: true}
	dark := frame(func(f device.Frame) bool { return f.Dark })
	if dark.Keys[0] != (device.Tile{}) {
		t.Fatal("dark frame shows a key")
	}
	generation = dark.Generation
	press()
	time.Sleep(200 * time.Millisecond)
	if presses.Load() != 1 {
		t.Fatal("a dark deck dispatched a press")
	}
	// Monitors off keeps it dark after unlock; unknown state never hides it.
	presences <- presence.State{Known: true, DisplayOff: true}
	frame(func(f device.Frame) bool { return f.Dark })
	presences <- presence.State{Known: true}
	back := frame(func(f device.Frame) bool { return !f.Dark && f.Keys[0].Label != "" })
	if back.Brightness != 70 {
		t.Fatal("restored brightness", back.Brightness)
	}
	presences <- presence.State{Locked: true}
	frame(func(f device.Frame) bool { return !f.Dark })
}
