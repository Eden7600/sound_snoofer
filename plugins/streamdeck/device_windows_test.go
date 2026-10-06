//go:build windows

package streamdeck

import (
	"context"
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
