//go:build windows

package app

import (
	"context"
	"encoding/json"
	"testing"
)

func TestDesktopBoundedActions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	d := Desktop{ctx: ctx, actions: make(chan UIAction, 1)}
	if err := d.Send(UIAction{Kind: "retry"}); err != nil {
		t.Fatal(err)
	}
	if err := d.Send(UIAction{Kind: "retry"}); err == nil {
		t.Fatal("unbounded dispatch")
	}
	<-d.actions
	cancel()
	if err := d.Send(UIAction{Kind: "retry"}); err == nil {
		t.Fatal("disconnected dispatch")
	}
}

func TestDesktopVersionedState(t *testing.T) {
	d := &Desktop{}
	if d.State(0) != nil {
		t.Fatal("state before first snapshot")
	}
	var sender viewSender
	state := artworkState()
	_, first := prepareFrame(t, &sender, state, false)
	d.applyFrame(first)
	initial := d.State(0)
	if initial == nil || initial.Version != 1 || initial.Artwork == nil || (*initial.Artwork)["clip"] != "first" || initial.AnimationVersion != 1 || len(d.Animation("clip")) != 2 {
		t.Fatal("incomplete desktop initialization", initial)
	}
	if d.State(initial.Version) != nil {
		t.Fatal("unchanged IPC returned state")
	}
	_, focus := prepareFrame(t, &sender, state, true)
	d.applyFrame(focus)
	if d.State(initial.Version) != nil {
		t.Fatal("focus invalidated the poll cursor")
	}
	state.Controls[0].Artwork = "new"
	_, changed := prepareFrame(t, &sender, state, false)
	d.applyFrame(changed)
	state.Controls[0].Progress.PositionMs = 150
	_, telemetry := prepareFrame(t, &sender, state, false)
	d.applyFrame(telemetry)
	late := d.State(initial.Version)
	if late == nil || late.Version != telemetry.Version || late.Artwork == nil || (*late.Artwork)["clip"] != "new" || late.State.Controls[0].Progress.PositionMs != 150 {
		t.Fatal("late caller missed artwork or telemetry", late)
	}
	for _, cursor := range []uint64{0, late.Version + 100} {
		if response := d.State(cursor); response == nil || response.Artwork == nil || (*response.Artwork)["clip"] != "new" {
			t.Fatal("fresh/unknown caller did not recover assets", cursor, response)
		}
	}
	state.Controls[0].Progress.PositionMs++
	_, telemetry = prepareFrame(t, &sender, state, false)
	d.applyFrame(telemetry)
	if next := d.State(late.Version); next == nil || next.Artwork != nil || next.AnimationVersion != initial.AnimationVersion {
		t.Fatal("telemetry repeated assets or animation generation", next)
	}
	state.Controls = nil
	_, cleared := prepareFrame(t, &sender, state, false)
	d.applyFrame(cleared)
	empty := d.State(telemetry.Version)
	if empty == nil || empty.Artwork == nil || len(*empty.Artwork) != 0 || d.Animation("clip") != nil || empty.AnimationVersion != cleared.Version {
		t.Fatal("desktop did not clear assets", empty)
	}
	encoded, err := json.Marshal(empty)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["Artwork"]) != "{}" {
		t.Fatal("IPC dropped empty-map clear", string(encoded))
	}
	if (*initial.Artwork)["clip"] != "first" {
		t.Fatal("later frames mutated a returned snapshot")
	}
}

func TestDesktopLegacyFrames(t *testing.T) {
	d := &Desktop{}
	state := artworkState()
	d.applyFrame(frame{State: &state})
	first := d.State(0)
	if first == nil || first.State.Controls[0].Artwork != "" || first.Artwork == nil || (*first.Artwork)["clip"] != "first" || state.Controls[0].Artwork != "first" {
		t.Fatal("legacy inline artwork not preserved immutably", first)
	}
	state.Controls[0].Artwork = ""
	d.applyFrame(frame{State: &state, Focus: true})
	second := d.State(first.Version)
	if second == nil || second.Artwork == nil || len(*second.Artwork) != 0 || second.State.Controls[0].Revision != 9 {
		t.Fatal("legacy replacement lost empty art or action revision", second)
	}
}
