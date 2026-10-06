package controller

import (
	"context"
	"path/filepath"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

func TestPlaybackMuteOwnsBothStatesAcrossRestartAndTransfer(t *testing.T) {
	c := config.Config{Studio: &config.Studio{Voice: &config.Voice{}}, Intent: &config.Intent{}}
	f := &mixerFake{s: model.Snapshot{Numbers: map[string]float32{"Bus[0].Mute": 1, "Bus[2].Mute": 0, "Bus[0].Gain": -12, "Bus[2].Gain": -5}}}
	p := routing.Plan{Topology: &routing.Topology{PlaybackTarget: "A1"}}
	m := &Mixer{Path: filepath.Join(t.TempDir(), "mute.json")}
	if err := m.Reconcile(f, c, p, f.s, true); err != nil {
		t.Fatal(err)
	}
	if f.s.Numbers["Bus[0].Mute"] != 0 {
		t.Fatal("native baseline defeated requested unmute")
	}
	// Simulate native drift and restart: the saved preference still wins.
	f.s.Numbers["Bus[0].Mute"] = 1
	m = &Mixer{Path: m.Path}
	if err := m.Reconcile(f, c, p, f.s, true); err != nil {
		t.Fatal(err)
	}
	if f.s.Numbers["Bus[0].Mute"] != 0 {
		t.Fatal("restart adopted native mute")
	}
	c.Intent.PlaybackMuted = true
	p.Topology.PlaybackTarget = "A3"
	if err := m.Reconcile(f, c, p, f.s, false); err != nil {
		t.Fatal(err)
	}
	if f.s.Numbers["Bus[2].Mute"] != 1 || f.s.Numbers["Bus[0].Mute"] != 0 {
		t.Fatal("unsafe transfer")
	}
	if err := m.Reconcile(f, c, p, f.s, true); err != nil {
		t.Fatal(err)
	}
	if f.s.Numbers["Bus[0].Mute"] != 1 {
		t.Fatal("obsolete baseline not restored")
	}
	if f.s.Numbers["Bus[0].Gain"] != -12 || f.s.Numbers["Bus[2].Gain"] != -5 {
		t.Fatal("routing changed gain")
	}
}

func TestPlaybackGainRejectsChangedDestination(t *testing.T) {
	base, b := topologyFixture(t)
	b.s.Assignments["A1"] = "Volt ASIO"
	b.s.Assignments["A2"] = "AirPods"
	b.s.Numbers["Bus[1].Gain"] = -10
	f := &mixerFake{s: b.s}
	base.Backend = f
	p, err := base.Plan()
	if err != nil {
		t.Fatal(err)
	}
	old := GainIdentity(&p, f.s, "playback")
	if old == "" {
		t.Fatal("settled playback unavailable")
	}
	f.s.Assignments["A2"] = "speakers"
	if err = base.Gain(context.Background(), "playback", old, 1, true); err == nil || len(f.writes) > 0 {
		t.Fatal("stale gain reached another device", err)
	}
	for _, bus := range []string{"A1", "A3", "A5"} {
		p.Topology.PlaybackTarget = bus
		f.s.Assignments[bus] = "Headphones"
		p.Topology.Operations = nil
		if GainTarget(&p, "playback") == "" || GainIdentity(&p, f.s, "playback") == "" {
			t.Fatal(bus)
		}
	}
}
