//go:build windows

package streamdeck

import (
	"context"
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

func TestCatalogueChangeCannotRetargetDisplayedKey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := snoofer.NewControls()
	pressed := make(chan string, 4)
	controls := []snoofer.Control{
		{ID: "test.a", Label: "A", Available: true, Operations: []string{"press"}},
		{ID: "test.b", Label: "B", Available: true, Operations: []string{"press"}},
	}
	publish := func() {
		t.Helper()
		if err := registry.Publish("test", controls, func(_ context.Context, r snoofer.Request) error { pressed <- r.ID; return nil }); err != nil {
			t.Fatal(err)
		}
	}
	publish()
	frames := make(chan device.Frame, 1)
	events := make(chan device.Event, 4)
	done := make(chan struct{})
	surface := func(ctx context.Context) (chan device.Frame, <-chan device.Event, <-chan struct{}) {
		go func() { <-ctx.Done(); close(done) }()
		return frames, events, done
	}
	layout := Layout{Home: "sounds", Pages: []Page{{ID: "sounds", Name: "Sounds", AutoControls: "test."}}}
	instance, err := startWithSurface(ctx, snoofer.Services{Controls: registry}, snoofer.MarshalSettings(Settings{Layout: layout}), surface)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		stopCtx, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		if err := instance.Stop(stopCtx); err != nil {
			t.Error(err)
		}
	}()
	var initial device.Frame
	select {
	case initial = <-frames:
	case <-time.After(time.Second):
		t.Fatal("no frame")
	}
	if initial.Keys[1].Label != "B" {
		t.Fatal("unexpected starting binding")
	}
	controls = append(controls, snoofer.Control{ID: "test.c", Label: "0", Available: true, Operations: []string{"press"}})
	publish()
	events <- device.Event{Encoder: -1, Key: 1, Press: true, Generation: initial.Generation}
	select {
	case id := <-pressed:
		if id != "test.b" {
			t.Fatalf("displayed B invoked %s", id)
		}
	case <-time.After(300 * time.Millisecond):
		// A refreshed frame may invalidate the old input; it must never retarget it.
	}
}
