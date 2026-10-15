//go:build windows

package streamdeck

import (
	"context"
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

func TestTemporaryPathReportRetirement(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := snoofer.NewControls()
	events := make(chan device.Event, 16)
	settings := Settings{
		Layout:  Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}}},
		Devices: map[string]Deck{"remembered": {Model: "plus"}},
	}
	instance, err := startWithSurface(ctx, snoofer.Services{Controls: registry}, snoofer.MarshalSettings(settings),
		idleSurface(make(chan device.Frame, 1), events, make(chan struct{})))
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
			if c.ID == reportID(id) {
				return c
			}
		}
		return snoofer.Control{}
	}
	wait := func(check func() bool) {
		t.Helper()
		for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
			if check() {
				return
			}
		}
		t.Fatal("connection reports did not reach expected state")
	}
	wait(func() bool { return find("remembered").Connection != nil })
	const path = "path:failed"
	events <- device.Event{Device: path, Error: "access denied"}
	wait(func() bool { c := find(path); return c.Connection != nil && c.Connection.LastError == "access denied" })
	events <- device.Event{Device: path, Retired: true}
	wait(func() bool { return find(path).ID == "" })
	if find("remembered").Connection == nil {
		t.Fatal("remembered serial report retired")
	}

	// Recovery with a serial replaces the temporary card.
	events <- device.Event{Device: path, Error: "busy"}
	wait(func() bool { return find(path).Connection != nil })
	events <- device.Event{Device: path, Retired: true}
	events <- device.Event{Device: "working", Serial: "working", Model: "plus", Connected: true}
	wait(func() bool {
		c := find("working")
		return c.Connection != nil && c.Connection.State == snoofer.ConnectionConnected
	})
	if find(path).ID != "" {
		t.Fatal("temporary card kept beside recovered serial")
	}

	// Recovery without a serial recreates the path as a connected report.
	events <- device.Event{Device: path, Error: "busy"}
	wait(func() bool { return find(path).Connection != nil })
	events <- device.Event{Device: path, Retired: true}
	events <- device.Event{Device: path, Model: "plus", Connected: true}
	wait(func() bool {
		c := find(path)
		return c.Connection != nil && c.Connection.State == snoofer.ConnectionConnected
	})
	if find(path).Connection.LastError != "" {
		t.Fatal("recreated path kept temporary failure")
	}
	// A temporary-report retirement must not erase active or serial state.
	events <- device.Event{Device: path, Retired: true}
	events <- device.Event{Device: "remembered", Error: "unplugged"}
	events <- device.Event{Device: "remembered", Retired: true}
	events <- device.Event{Device: "working", Error: "barrier"}
	wait(func() bool { c := find("working"); return c.Connection != nil && c.Connection.LastError == "barrier" })
	if c := find("remembered"); c.Connection == nil || c.Connection.LastError != "unplugged" {
		t.Fatal("serial report history was retired", c)
	}
	if find(path).Connection == nil || find("working").Connection == nil {
		t.Fatal("active reports retired")
	}
}
