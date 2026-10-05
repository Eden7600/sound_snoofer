package controller

import (
	"fmt"
	"path/filepath"
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
	"testing"
)

type mixerFake struct {
	s      model.Snapshot
	writes []string
	fail   bool
}

func (f *mixerFake) Snapshot() (model.Snapshot, error) { return f.s, nil }
func (f *mixerFake) Set(string, model.Device) error {
	return fmt.Errorf("unexpected device assignment")
}
func (f *mixerFake) SetMixer(p string, v float32) error {
	f.writes = append(f.writes, p)
	if f.fail {
		return fmt.Errorf("failed")
	}
	f.s.Numbers[p] = v
	return nil
}
func TestMuteOwnershipTransferAndComposition(t *testing.T) {
	c := config.Config{Studio: &config.Studio{Voice: &config.Voice{}}, Intent: &config.Intent{Enabled: true, Source: "desk", MicMuted: true, PlaybackMuted: true}}
	f := &mixerFake{s: model.Snapshot{Numbers: map[string]float32{"Strip[0].Mute": 0, "Strip[1].Mute": 0, "Strip[6].Mute": 1, "Bus[0].Mute": 0, "Bus[1].Mute": 0}}}
	m := &Mixer{Path: filepath.Join(t.TempDir(), "mute.json")}
	p := routing.Plan{Topology: &routing.Topology{Voice: &routing.VoiceStatus{Strip: 0}, PlaybackTarget: "A2"}}
	if e := m.Reconcile(f, c, p, f.s, true); e != nil {
		t.Fatal(e)
	}
	if f.s.Numbers["Strip[0].Mute"] != 1 || f.s.Numbers["Bus[1].Mute"] != 1 {
		t.Fatal("mute not applied")
	}
	p.Topology.Voice.Strip = 1
	p.Topology.PlaybackTarget = "A1"
	c.Intent.BusMuted[1] = true
	if e := m.Reconcile(f, c, p, f.s, false); e != nil {
		t.Fatal(e)
	}
	if f.s.Numbers["Strip[0].Mute"] != 1 || f.s.Numbers["Strip[1].Mute"] != 1 {
		t.Fatal("old mute released before route settled")
	}
	if e := m.Reconcile(f, c, p, f.s, true); e != nil {
		t.Fatal(e)
	}
	if f.s.Numbers["Strip[0].Mute"] != 0 || f.s.Numbers["Bus[1].Mute"] != 1 {
		t.Fatal("ownership transfer/composition")
	}
	c.Intent.MicMuted = false
	c.Intent.PlaybackMuted = false
	c.Intent.BusMuted[1] = false
	// A new owner restores the durable baseline after restart.
	m = &Mixer{Path: m.Path}
	if e := m.Reconcile(f, c, p, f.s, true); e != nil {
		t.Fatal(e)
	}
	if f.s.Numbers["Strip[6].Mute"] != 1 || f.s.Numbers["Strip[1].Mute"] != 0 {
		t.Fatal("manual baseline lost")
	}
}
func TestMuteFailureBlocksNewPath(t *testing.T) {
	m := &Mixer{}
	f := &mixerFake{fail: true, s: model.Snapshot{Numbers: map[string]float32{"Bus[0].Mute": 0}}}
	c := config.Config{Studio: &config.Studio{Voice: &config.Voice{}}, Intent: &config.Intent{PlaybackMuted: true}}
	p := routing.Plan{Topology: &routing.Topology{PlaybackTarget: "A1"}}
	if m.Reconcile(f, c, p, f.s, false) == nil {
		t.Fatal("failed mute accepted")
	}
}
