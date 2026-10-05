//go:build windows

package streamdeck

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

func TestEditorSaveAndInputGenerations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := snoofer.NewControls()
	var presses atomic.Int32
	provider := func() {
		if err := registry.Publish("test", []snoofer.Control{{ID: "test.press", Label: "Press", Available: true, Operations: []string{"press"}}}, func(context.Context, snoofer.Request) error { presses.Add(1); return nil }); err != nil {
			t.Fatal(err)
		}
	}
	provider()
	layout := Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}}
	layout.Pages[0].Keys[0] = Binding{Control: "test.press", Label: "Saved press"}
	frames := make(chan device.Frame, 1)
	events := make(chan device.Event, 16)
	done := make(chan struct{})
	surface := func(ctx context.Context) (chan device.Frame, <-chan device.Event, <-chan struct{}) {
		go func() { <-ctx.Done(); close(done) }()
		return frames, events, done
	}
	var fail atomic.Bool
	fail.Store(true)
	saved := make(chan Settings, 1)
	services := snoofer.Services{Controls: registry, SaveSettings: func(_ string, _, data json.RawMessage) error {
		if fail.Load() {
			return errors.New("save denied")
		}
		var settings Settings
		if err := json.Unmarshal(data, &settings); err != nil {
			return err
		}
		saved <- settings
		return nil
	}}
	instance, err := startWithSurface(ctx, services, snoofer.MarshalSettings(Settings{Layout: layout}), surface)
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
	control := func(id string) snoofer.Control {
		for _, c := range registry.Snapshot() {
			if c.ID == "streamdeck."+id {
				return c
			}
		}
		return snoofer.Control{}
	}
	wait := func(check func() bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if check() {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("actor did not reach expected state")
	}
	wait(func() bool { return control("name").Revision != 0 })
	command := func(id, op, value string) {
		t.Helper()
		c := control(id)
		if err := registry.Dispatch(ctx, snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: op, Value: value}); err != nil {
			t.Fatal(err)
		}
		wait(func() bool { return control(id).Revision != c.Revision })
	}
	frame := func(name, value string) device.Frame {
		t.Helper()
		deadline := time.NewTimer(2 * time.Second)
		defer deadline.Stop()
		for {
			select {
			case f := <-frames:
				if f.Dials[5].Value == name && (value == "" || f.Keys[0].Value == value) {
					return f
				}
			case <-deadline.C:
				t.Fatal("frame not rendered")
				return device.Frame{}
			}
		}
	}
	initial := frame("A", "")
	command("name", "set", "Renamed")
	command("save", "press", "")
	if !strings.Contains(control("status").Value, "save denied") {
		t.Fatal(control("status"))
	}
	frame("A", "") // A failed save must not change the active layout.
	fail.Store(false)
	command("save", "press", "")
	committed := <-saved
	if committed.Layout.Pages[0].Name != "Renamed" {
		t.Fatal("lost draft")
	}
	current := frame("Renamed", "")
	events <- device.Event{Encoder: -1, Key: 0, Press: true, Generation: initial.Generation}
	events <- device.Event{Encoder: -1, Key: 0, Press: true, Generation: current.Generation}
	wait(func() bool { return presses.Load() == 1 })
	events <- device.Event{Encoder: 5, Delta: 1, Generation: current.Generation}
	pageB := frame("B", "")
	events <- device.Event{Encoder: -1, Key: 0, Press: true, Generation: current.Generation}
	events <- device.Event{Encoder: 5, Press: true, Generation: pageB.Generation}
	current = frame("Renamed", "")
	if presses.Load() != 1 {
		t.Fatal("old page input replayed")
	}
	registry.Remove("test")
	frame("Renamed", "Unavailable")
	provider()
	restored := frame("Renamed", "")
	events <- device.Event{Encoder: -1, Key: 0, Press: true, Generation: current.Generation}
	events <- device.Event{Encoder: -1, Key: 0, Press: true, Generation: restored.Generation}
	wait(func() bool { return presses.Load() == 2 })
	command("page", "set", "b")
	command("earlier", "press", "")
	command("save", "press", "")
	if (<-saved).Layout.Pages[0].ID != "b" {
		t.Fatal("reorder changed page identity")
	}
}
