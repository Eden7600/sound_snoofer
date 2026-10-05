package audio

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/snoofer"
)

func TestSemanticProfileEdits(t *testing.T) {
	s := control.State{Profile: "VR", VRConfigured: true, Revision: 7, Intent: &config.Intent{Source: "lav", Mode: "element", Monitor: "off", VRProfile: &config.ProfileChoices{Source: "auto", Mode: "direct", Monitor: "off"}}}
	s.ActiveIntent = s.Intent.Clone()
	s.ActiveIntent.Mode = "direct"
	a, err := action(s, snoofer.Request{ID: "audio.mode", Operation: "press"})
	if err != nil || a.Row != "vr-profile-mode" || a.Value != "element" {
		t.Fatal(a, err)
	}
	normal, err := action(s, snoofer.Request{ID: "audio.normal-source", Operation: "set", Value: "webcam"})
	if err != nil {
		t.Fatal(err)
	}
	next := s.Intent.Clone()
	if err := control.EditIntent(next, normal); err != nil {
		t.Fatal(err)
	}
	if next.Source != "webcam" || next.VRProfile.Source != "auto" {
		t.Fatal(next)
	}
	subdued := false
	for _, c := range controls(s) {
		if c.ID == "audio.normal-source" {
			subdued = c.Subdued && c.Available
		}
	}
	if !subdued {
		t.Fatal("Normal must remain editable while overridden")
	}
}
func TestUnknownVRRetainsActivation(t *testing.T) {
	i := &Instance{actions: make(chan control.Action, 8)}
	p := VRPolicy{Devices: config.VR{Input: 4}, Microphones: []string{"normal"}, Playback: []config.Candidate{{Driver: "normal"}}, Choices: config.ProfileChoices{Source: "auto", Mode: "direct", Monitor: "off"}, Known: true, Running: true}
	if err := i.SetVRPolicy(p); err != nil {
		t.Fatal(err)
	}
	p.Known = false
	p.Running = false
	if err := i.SetVRPolicy(p); err != nil {
		t.Fatal(err)
	}
	if !i.policySnapshot().Running {
		t.Fatal("unknown forced profile transition")
	}
}

func TestPartialNativeStartCleansUp(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "snoofer.json")
	registry := snoofer.NewControls()
	settings := Settings{Config: config.DefaultBytes(), StatePath: "audio", DLL: filepath.Join(filepath.Dir(path), "missing.dll")}
	instance, err := start(ctx, snoofer.Services{Controls: registry, Path: path}, snoofer.MarshalSettings(settings), nil)
	if err == nil || instance != nil {
		t.Fatal("invalid DLL started", instance, err)
	}
	if len(registry.Snapshot()) != 0 {
		t.Fatal("failed startup left controls")
	}
}
func TestMicDialPressDoesNotChangeSource(t *testing.T) {
	s := control.State{Intent: &config.Intent{Source: "desk", Enabled: true}}
	a, err := action(s, snoofer.Request{ID: "audio.gain-mic", Operation: "press"})
	if err != nil || a.Row != "mic-mute" || a.Value != "true" {
		t.Fatal(a, err)
	}
	if s.Intent.Source != "desk" {
		t.Fatal("changed source")
	}
}
