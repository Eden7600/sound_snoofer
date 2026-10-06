package audio

import (
	"slices"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/snoofer"
)

func TestMicStackSemanticControls(t *testing.T) {
	s := control.State{Profile: "VR", VRConfigured: true, MicOptions: []string{"desk", "webcam", "off"}, VRSourceOptions: []string{"auto", "off", "vr:h"}, Intent: &config.Intent{Enabled: false, Source: "webcam", MicMuted: true, Mode: "direct", Monitor: "off", VRProfile: &config.ProfileChoices{Source: "vr:h", Mode: "direct", Monitor: "off"}}}
	s.ActiveIntent = s.Intent.Clone()
	s.ActiveIntent.Source = "off"
	byID := map[string]snoofer.Control{}
	for _, c := range controls(s) {
		byID[c.ID] = c
	}
	for _, id := range []string{"audio.source", "audio.normal-source", "audio.vr-profile-source"} {
		c := byID[id]
		if slices.Contains(c.Options, "off") || !slices.Contains(c.Options, "auto") || !c.Available {
			t.Fatal(id, c)
		}
	}
	if byID["audio.source"].Value != "vr:h" || byID["audio.normal-source"].Value != "webcam" {
		t.Fatal("displayed resolved Off instead of retained target")
	}
	master := byID["audio.mic-stack"]
	if master.Value != "Off" || master.Subdued || !master.Available {
		t.Fatal(master)
	}
	a, err := action(s, snoofer.Request{ID: master.ID, Operation: "press"})
	if err != nil || a.Row != "mic-stack" || a.Value != "true" {
		t.Fatal(a, err)
	}
	if byID["audio.mic-mute"].Value != "On" {
		t.Fatal("mute lost")
	}
}

func TestDisabledStackDoesNotHideNativeError(t *testing.T) {
	s := control.State{Connected: true, Error: "native write failed", Intent: &config.Intent{Enabled: false, Source: "webcam"}}
	for _, c := range controls(s) {
		if c.ID == "audio.mic-stack" && c.Status != s.Error {
			t.Fatal(c)
		}
	}
}

func TestMicControlsHiddenOnSurfacesWhenStackOff(t *testing.T) {
	intent := &config.Intent{Enabled: false, Source: "desk", Mode: "direct", Monitor: "off", Recording: &config.RecordingChoices{}}
	s := control.State{Connected: true, Live: true, MicOptions: []string{"desk", "off"}, Intent: intent, ActiveIntent: intent.Clone()}
	hidden := map[string]bool{}
	byID := map[string]snoofer.Control{}
	for _, c := range controls(s) {
		byID[c.ID] = c
		hidden[c.ID] = c.Hidden
	}
	for _, id := range []string{"audio.mic-mute", "audio.source", "audio.mode", "audio.monitor", "audio.record-mic", "audio.record-tap", "audio.normal-source"} {
		if _, ok := byID[id]; !ok {
			t.Fatalf("%s not published", id)
		}
		if !hidden[id] {
			t.Errorf("%s visible with the mic stack off", id)
		}
	}
	if hidden["audio.mic-stack"] || hidden["audio.speaker-mute"] || hidden["audio.record-computer"] {
		t.Fatal("non-mic control hidden")
	}
	if !byID["audio.mode"].Available {
		t.Fatal("hiding changed availability")
	}
	s.Intent = &config.Intent{Enabled: true, Source: "desk", Mode: "direct", Monitor: "off", Recording: &config.RecordingChoices{}}
	s.ActiveIntent = s.Intent.Clone()
	for _, c := range controls(s) {
		if c.Hidden {
			t.Errorf("%s hidden with the mic stack on", c.ID)
		}
	}
}
