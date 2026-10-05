package routing

import (
	"slices"
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"testing"
)

func TestVRRuntimeGatingPreferenceAndOff(t *testing.T) {
	c, s := voiceFixture(t)
	c.VR = &config.VR{Input: 4, Headsets: []config.Headset{{ID: "headset", Microphone: "^VR Mic$", Playback: "^VR Phones$"}}}
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	c.Intent = c.VoiceIntent()
	c.Intent.PreferVRMic = true
	c.Intent.PreferVRPlayback = true
	s.Devices = append(s.Devices, model.Device{Name: "VR Mic", Direction: "input", Driver: "wdm", Available: true}, model.Device{Name: "VR Phones", Direction: "output", Driver: "wdm", Available: true})
	if slices.Contains(MicrophoneOptions(c, s), "vr:headset") {
		t.Fatal("unknown SteamVR allowed headset")
	}
	s.SteamVR = &model.ProcessStatus{Known: true, Running: true}
	p, e := Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	if p.Topology.Voice.Strip != 3 || p.Topology.Voice.Effective != "vr:headset" {
		t.Fatal(p.Topology.Voice)
	}
	if !slices.Contains(PlaybackOptions(c, s), "VR Phones") {
		t.Fatal("missing headset playback")
	}
	applyPlan(&s, p)
	p, e = Build(c, s)
	if e != nil || p.HasChanges() {
		t.Fatal("stable headset reassigned", e, p)
	}
	s.SteamVR.Running = false
	p, e = Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	if p.Topology.Voice.Effective == "vr:headset" {
		t.Fatal("headset persisted after VR exit")
	}
	s.SteamVR.Running = true
	c.Intent.Source = "off"
	c.Intent.Enabled = false
	p, e = Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	if p.Topology.Voice.Strip != -1 {
		t.Fatal("Off overridden")
	}
	applyPlan(&s, p)
	if s.Assignments["input:4"] != "" {
		t.Fatal("Off retained VR mic")
	}
}
func TestVRAmbiguousDeviceCannotLeakThroughGenericMatcher(t *testing.T) {
	c, s := voiceFixture(t)
	c.VR = &config.VR{Headsets: []config.Headset{{ID: "vr", Playback: "^VR"}}}
	c.Validate()
	s.SteamVR = &model.ProcessStatus{Known: true, Running: true}
	s.Devices = append(s.Devices, model.Device{Name: "VR 1", Direction: "output", Driver: "wdm", Available: true}, model.Device{Name: "VR 2", Direction: "output", Driver: "wdm", Available: true})
	filtered := VRDevices(c, s)
	for _, d := range filtered.Devices {
		if (d.Name == "VR 1" || d.Name == "VR 2") && d.Available {
			t.Fatal("ambiguous VR endpoint eligible")
		}
	}
}
