//go:build windows

package streamdeck

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

func TestEditorRegionClipAndMakeHome(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := snoofer.NewControls()
	layout := Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A", Regions: []Region{{Source: "a.s", First: 0, Last: 1}}}}}
	done := make(chan struct{})
	surface := func(ctx context.Context) (chan device.Frame, <-chan device.Event, <-chan struct{}) {
		go func() { <-ctx.Done(); close(done) }()
		return make(chan device.Frame, 1), make(chan device.Event), done
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
	view := func() EditorView {
		t.Helper()
		var v EditorView
		if err := json.Unmarshal(control("preview").ViewData, &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	command := func(id, op, value string) {
		t.Helper()
		c := control(id)
		if err := registry.Dispatch(ctx, snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: op, Value: value}); err != nil {
			t.Fatal(err)
		}
		wait(func() bool { return control(id).Revision != c.Revision })
	}
	wait(func() bool { return control("region-clip").Revision != 0 })
	command("home", "press", "")
	if v := view(); v.Dirty || !v.Home {
		t.Fatal("Make Home on Home marked the draft unsaved", v.Dirty, v.Home)
	}
	command("region-clip", "set", "0,on")
	if v := view(); !v.Dirty || !v.Regions[0].Clip {
		t.Fatal("clip edit not applied", v.Dirty, v.Regions)
	}
	command("region-clip", "set", "3,on")
	if v := view(); !v.Regions[0].Clip || !strings.Contains(control("status").Value, "unknown region") {
		t.Fatal("unknown region not refused", v.Regions, control("status").Value)
	}
	stale := control("region-clip")
	command("cancel", "press", "")
	if err := registry.Dispatch(ctx, snoofer.Request{ID: stale.ID, Revision: stale.Revision, Operation: "set", Value: "0,on"}); err == nil {
		wait(func() bool { return control("status").Value == "Editor changed; try again" || view().Regions[0].Clip })
	}
	if v := view(); v.Dirty || v.Regions[0].Clip {
		t.Fatal("stale or discarded clip edit kept", v.Dirty, v.Regions)
	}
}
