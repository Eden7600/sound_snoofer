//go:build windows

package streamdeck

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

func TestStatusKeysAndEditor(t *testing.T) {
	for _, publishFirst := range []bool{true, false} {
		name := "provider after startup"
		if publishFirst {
			name = "provider before startup"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			registry := snoofer.NewControls()
			requests := make(chan snoofer.Request, 8)
			provider := func(value string) {
				t.Helper()
				err := registry.Publish("test", []snoofer.Control{
					{ID: "test.status", Label: "Status", Kind: "status", Value: value, Available: true},
					// Status wins even if a provider advertises actions.
					{ID: "test.held-status", Kind: "status", Operations: []string{"press", "hold", "adjust"}, Available: true},
					{ID: "test.press", Kind: "command", Operations: []string{"press"}, Available: true},
					{ID: "test.gain", Kind: "numeric", Operations: []string{"adjust"}, Available: true},
					{ID: "test.text", Kind: "text", Operations: []string{"set", "adjust"}, Available: true},
					{ID: "test.connection", Kind: "connection", Available: true},
				}, func(_ context.Context, r snoofer.Request) error { requests <- r; return nil })
				if err != nil {
					t.Fatal(err)
				}
			}
			if publishFirst {
				provider("First")
			}
			layout := Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}}}
			layout.Pages[0].Keys[0] = Binding{Control: "test.status"}
			layout.Pages[0].Keys[1] = Binding{Control: "test.press"}
			layout.Pages[0].Keys[2] = Binding{Control: "test.held-status"}
			frames := make(chan device.Frame, 1)
			events := make(chan device.Event, 16)
			saved := make(chan Settings, 8)
			services := snoofer.Services{Controls: registry, SaveSettings: func(_ string, _, data json.RawMessage) error {
				var settings Settings
				if err := json.Unmarshal(data, &settings); err != nil {
					return err
				}
				saved <- settings
				return nil
			}}
			settings := Settings{Layout: layout, Devices: map[string]Deck{testDeck: {Model: "plus-xl"}}}
			instance, err := startWithSurface(ctx, services, snoofer.MarshalSettings(settings), fakeSurface(frames, events, make(chan struct{})))
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
			if !publishFirst {
				provider("First")
			}
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
				for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
					if check() {
						return
					}
				}
				t.Fatal("editor did not reach expected state")
			}
			command := func(id, op, value string) {
				t.Helper()
				c := control(id)
				if err := registry.Dispatch(ctx, snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: op, Value: value}); err != nil {
					t.Fatal(err)
				}
				wait(func() bool { return control(id).Revision != c.Revision })
			}
			frame := func(value string) device.Frame {
				t.Helper()
				deadline := time.NewTimer(2 * time.Second)
				defer deadline.Stop()
				for {
					select {
					case f := <-frames:
						if f.Keys[0].Value == value {
							return f
						}
					case <-deadline.C:
						t.Fatal("status frame not rendered")
						return device.Frame{}
					}
				}
			}
			current := frame("First")
			wait(func() bool { return control("binding").Revision != 0 })
			var pressRevision uint64
			for _, c := range registry.Snapshot() {
				if c.ID == "test.press" {
					pressRevision = c.Revision
				}
			}
			key := func(n int, down bool) {
				events <- device.Event{Device: testDeck, Key: n, Encoder: -1, Press: down, Release: !down, Generation: current.Generation}
			}
			key(0, true)
			key(0, false)
			key(2, true)
			time.Sleep(holdAfter + 200*time.Millisecond)
			key(2, false)
			// This following event proves the actor processed the status release too.
			key(1, true)
			select {
			case r := <-requests:
				if r.ID != "test.press" || r.Operation != "press" || r.Revision != pressRevision {
					t.Fatalf("status input dispatched or neighbor changed: %+v", r)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("actionable neighbor did not dispatch")
			}
			if note := control("status").Value; note != "" {
				t.Fatalf("status input produced an editor error: %s", note)
			}
			provider("Second")
			frame("Second")
			if len(requests) != 0 {
				t.Fatal("status input dispatched another request")
			}

			// Use a blank key so the current-binding fallback cannot hide an omitted option.
			command("slot", "set", "Key 4")
			for _, id := range []string{"test.status", "test.held-status", "test.press"} {
				if !slices.Contains(control("binding").Options, id) {
					t.Fatal("key option missing", id)
				}
			}
			for _, id := range []string{"test.text", "test.connection", "test.gain"} {
				if slices.Contains(control("binding").Options, id) {
					t.Fatal("incompatible key option", id)
				}
			}
			command("binding", "set", "test.status")
			command("save", "press", "")
			select {
			case committed := <-saved:
				want := layout.clone()
				want.Pages[0].Keys[3] = Binding{Control: "test.status", Label: "Status"}
				if !reflect.DeepEqual(committed.Layout.Pages, want.Pages) {
					t.Fatal("saving the status key changed another binding", committed.Layout.Pages)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("status binding was not saved", control("status").Value)
			}
			command("slot", "set", "Dial 1")
			for _, id := range []string{"test.status", "test.held-status", "test.text", "test.connection", "test.press"} {
				if slices.Contains(control("binding").Options, id) {
					t.Fatal("incompatible dial option", id)
				}
			}
			if !slices.Contains(control("binding").Options, "test.gain") {
				t.Fatal("adjustable dial option missing")
			}
		})
	}
}
