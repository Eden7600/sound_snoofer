//go:build windows

package streamdeck

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

func TestDeviceReportSeparateFromLayoutStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	registry := snoofer.NewControls()
	frames := make(chan device.Frame, 8)
	events := make(chan device.Event, 16)
	done := make(chan struct{})
	surface := func(ctx context.Context) (chan device.Frame, <-chan device.Event, <-chan struct{}) {
		go func() { <-ctx.Done(); close(done) }()
		return frames, events, done
	}
	layout := Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}}}
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
	find := func(id string) snoofer.Control {
		for _, c := range registry.Snapshot() {
			if c.ID == id {
				return c
			}
		}
		return snoofer.Control{}
	}
	wait := func(ok func(snoofer.Control) bool) snoofer.Control {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if c := find("streamdeck.app-device"); c.Connection != nil && ok(c) {
				return c
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("device report not reached; last %+v", find("streamdeck.app-device"))
		return snoofer.Control{}
	}
	go func() {
		for range frames { // Drain rendered frames.
		}
	}()
	wait(func(c snoofer.Control) bool { return c.Connection.State == snoofer.ConnectionDisconnected })
	events <- device.Event{Connected: true, Serial: "A00WA4012345"}
	report := wait(func(c snoofer.Control) bool { return c.Connection.State == snoofer.ConnectionConnected })
	if report.Connection.Details[0].Value != "A00WA4012345" || report.Kind != "connection" || len(report.Operations) != 0 {
		t.Fatalf("connected report %+v", report.Connection)
	}
	if status := find("streamdeck.status"); status.Value == "Connected" {
		t.Fatal("device state leaked into layout status")
	}
	events <- device.Event{Error: "Stream Deck disconnected"}
	report = wait(func(c snoofer.Control) bool { return c.Connection.State == snoofer.ConnectionDisconnected })
	if report.Connection.LastError != "Stream Deck disconnected" {
		t.Fatalf("disconnect report %+v", report.Connection)
	}
	if status := find("streamdeck.status"); status.Value == "Stream Deck disconnected" {
		t.Fatal("device error leaked into layout status")
	}
	events <- device.Event{Connected: true, Serial: "A00WA4012345"}
	report = wait(func(c snoofer.Control) bool { return c.Connection.State == snoofer.ConnectionConnected })
	if len(report.Connection.Details) != 2 || report.Connection.Details[1].Value != "1" || report.Connection.LastError == "" {
		t.Fatalf("reconnect report %+v", report.Connection)
	}
}

func TestHiddenBindingIgnoresInput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	registry := snoofer.NewControls()
	var presses atomic.Int32
	publish := func(hidden bool) {
		err := registry.Publish("test", []snoofer.Control{{ID: "test.mic", Label: "Mic", Icon: "mic-mute", Available: true, Hidden: hidden, Operations: []string{"press"}}},
			func(context.Context, snoofer.Request) error { presses.Add(1); return nil })
		if err != nil {
			t.Fatal(err)
		}
	}
	publish(true)
	frames := make(chan device.Frame, 8)
	events := make(chan device.Event, 16)
	done := make(chan struct{})
	surface := func(ctx context.Context) (chan device.Frame, <-chan device.Event, <-chan struct{}) {
		go func() { <-ctx.Done(); close(done) }()
		return frames, events, done
	}
	layout := Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}}}
	layout.Pages[0].Keys[0] = Binding{Control: "test.mic", Label: "Mic"}
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
	next := func(blank bool) device.Frame {
		t.Helper()
		deadline := time.After(2 * time.Second)
		for {
			select {
			case f := <-frames:
				if (f.Keys[0] == device.Tile{}) == blank {
					return f
				}
			case <-deadline:
				t.Fatalf("no frame with blank=%v", blank)
				return device.Frame{}
			}
		}
	}
	hiddenFrame := next(true)
	events <- device.Event{Encoder: -1, Key: 0, Press: true, Generation: hiddenFrame.Generation}
	time.Sleep(100 * time.Millisecond)
	if presses.Load() != 0 {
		t.Fatal("input on a hidden binding dispatched")
	}
	publish(false)
	shown := next(false)
	events <- device.Event{Encoder: -1, Key: 0, Press: true, Generation: shown.Generation}
	deadline := time.Now().Add(2 * time.Second)
	for presses.Load() != 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if presses.Load() != 1 {
		t.Fatal("binding did not return when unhidden")
	}
}
