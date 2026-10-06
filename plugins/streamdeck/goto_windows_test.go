//go:build windows

package streamdeck

import (
	"context"
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

func TestGotoPageKeys(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := snoofer.NewControls()
	layout := Layout{Home: "home", Pages: []Page{{ID: "home", Name: "Home"}, {ID: "sounds", Name: "Soundboard"}, {ID: "lights", Name: "Lights"}}}
	layout.Pages[0].Keys[0] = Binding{Control: gotoPrefix + "lights", Label: "Go to Lights"}
	layout.Pages[0].Keys[1] = Binding{Control: gotoPrefix + "gone", Label: "Go to Gone"}
	layout.Pages[2].Keys[8] = Binding{Control: gotoPrefix + "home", Label: "Go to Home"}
	frames := make(chan device.Frame, 1)
	events := make(chan device.Event, 16)
	done := make(chan struct{})
	surface := func(ctx context.Context) (chan device.Frame, <-chan device.Event, <-chan struct{}) {
		go func() { <-ctx.Done(); close(done) }()
		return frames, events, done
	}
	instance, err := startWithSurface(ctx, snoofer.Services{Controls: registry}, snoofer.MarshalSettings(Settings{Layout: layout}), surface)
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
	frame := func(page string, ready func(device.Frame) bool) device.Frame {
		t.Helper()
		deadline := time.NewTimer(2 * time.Second)
		defer deadline.Stop()
		for {
			select {
			case f := <-frames:
				if f.Dials[5].Value == page && (ready == nil || ready(f)) {
					return f
				}
			case <-deadline.C:
				t.Fatal("page not shown:", page)
				return device.Frame{}
			}
		}
	}
	home := frame("Home", nil)
	if home.Keys[0].Label != "Lights" || home.Keys[0].Icon != "deck-page" || home.Keys[1].Value != "N/A" {
		t.Fatalf("go-to keys %+v %+v", home.Keys[0], home.Keys[1])
	}
	events <- device.Event{Encoder: -1, Key: 1, Press: true, Generation: home.Generation} // Deleted page: inert.
	events <- device.Event{Encoder: -1, Key: 0, Press: true, Generation: home.Generation}
	lights := frame("Lights", nil)
	if lights.Keys[8].Label != "Home" || lights.Keys[8].Value != "" {
		t.Fatalf("home key %+v", lights.Keys[8])
	}
	// The registry is published just after the frame is sent.
	marked := func() bool {
		for _, c := range registry.Snapshot() {
			if c.ID == gotoPrefix+"lights" {
				return c.Value == "Here"
			}
		}
		return false
	}
	for deadline := time.Now().Add(time.Second); !marked(); time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("shown page not marked Here")
		}
	}
	events <- device.Event{Encoder: -1, Key: 8, Press: true, Generation: lights.Generation}
	frame("Home", nil)
}
