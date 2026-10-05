package routing

import (
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"testing"
)

func TestIndependentProfilesAndExplicitFallback(t *testing.T) {
	c, s := voiceFixture(t)
	c.Profiles = &config.Profiles{Microphones: []string{"lav", "webcam"}}
	c.Intent = c.VoiceIntent()
	c.Intent.Source = "lav"
	c.Intent.MicMuted = true
	policy := config.ProfilePolicy{
		Devices:     config.VR{Input: 4, Headsets: []config.Headset{{ID: "headset", Label: "Headset", Microphone: "^Headset mic$", Playback: "^Headset out$"}}},
		Microphones: []string{"vr:headset"}, Playback: []config.Candidate{{Driver: "wdm", Pattern: "^Headset out$"}},
		Choices: config.ProfileChoices{Source: "auto", Mode: "direct", Monitor: "off"}, Known: true, Running: true,
	}
	if err := policy.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Policy = func() *config.ProfilePolicy { return &policy }
	s.Devices = append(s.Devices, model.Device{Name: "Headset mic", Direction: "input", Driver: "wdm", Available: true}, model.Device{Name: "Headset out", Direction: "output", Driver: "wdm", Available: true})
	effective := ProfileConfig(c, s)
	if effective.Intent.Source != "vr:headset" || !effective.Intent.MicMuted || effective.ProfilePlayback.Name != "Headset out" {
		t.Fatal(effective.Intent, effective.ProfilePlayback)
	}
	p, err := Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if p.Topology.Voice.Effective != "vr:headset" {
		t.Fatal(p.Topology.Voice)
	}
	s.Devices[len(s.Devices)-1].Available = false
	s.Devices[len(s.Devices)-2].Available = false
	effective = ProfileConfig(c, s)
	if effective.ProfilePlayback != nil || !effective.ProfileMicMissing {
		t.Fatal("implicit Normal fallback")
	}
	policy.Microphones = append(policy.Microphones, "normal")
	policy.Playback = append(policy.Playback, config.Candidate{Driver: "normal"})
	effective = ProfileConfig(c, s)
	if effective.Intent.Source != "lav" || effective.ProfilePlayback == nil {
		t.Fatal("explicit fallback failed")
	}
	policy.Running = false
	c.Intent.Source = "webcam"
	effective = ProfileConfig(effective, s) // A previous derived config must not become saved preferences.
	if effective.Intent.Source != "webcam" {
		t.Fatal(effective.Intent.Source)
	}
	if !c.Intent.MicMuted {
		t.Fatal("profile transition cleared shared mute")
	}
}
func TestProfileAmbiguityAndOff(t *testing.T) {
	c, s := voiceFixture(t)
	c.Profiles = &config.Profiles{Microphones: []string{"lav", "webcam"}}
	policy := config.ProfilePolicy{Devices: config.VR{Input: 4, Headsets: []config.Headset{{ID: "h", Microphone: "Headset"}}}, Microphones: []string{"vr:h"}, Playback: []config.Candidate{{Driver: "normal"}}, Choices: config.ProfileChoices{Source: "auto", Mode: "direct", Monitor: "off"}, Known: true, Running: true}
	if err := policy.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Policy = func() *config.ProfilePolicy { return &policy }
	for n := 0; n < 2; n++ {
		s.Devices = append(s.Devices, model.Device{Name: "Headset", Driver: "wdm", Direction: "input", Available: true})
	}
	result := ProfileConfig(c, s)
	if !result.ProfileMicMissing {
		t.Fatal("selected ambiguous headset")
	}
	policy.Choices.Source = "off"
	result = ProfileConfig(c, s)
	if result.Intent.MicActive() || result.ProfileMicMissing || result.ProfilePlayback == nil {
		t.Fatal("Off affected playback or became failure")
	}
}

func TestNormalClearsRetiredVRMonitoring(t *testing.T) {
	c, s := voiceFixture(t)
	c.Profiles = &config.Profiles{Microphones: []string{"lav", "webcam"}}
	c.VR = &config.VR{Input: 4, Headsets: []config.Headset{{ID: "h", Microphone: "^Headset$"}}}
	if err := c.VR.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Assignments["input:4"] = "Headset"
	s.Numbers["Strip[3].A2"] = 1
	p, err := Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	clearedInput, clearedMonitor := false, false
	for _, op := range p.Topology.Operations {
		if op.Target == "input:4" && op.Device != nil && op.Device.Name == "" {
			clearedInput = true
		}
		if op.Parameter == "Strip[3].A2" && op.Value == 0 {
			clearedMonitor = true
		}
	}
	if !clearedInput || !clearedMonitor {
		t.Fatal("retired VR routing retained", clearedInput, clearedMonitor)
	}
}

func TestProfileDoesNotAssignAnEmptyPlaybackTarget(t *testing.T) {
	c, s := voiceFixture(t)
	c.Profiles = &config.Profiles{Microphones: []string{"lav", "webcam"}}
	for _, target := range []string{"A2", "A3", "A4", "A5"} {
		s.Assignments[target] = "unmanaged output"
	}
	p, err := Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range p.Topology.Operations {
		if op.Device != nil && op.Target == "" {
			t.Fatal("invalid empty playback assignment")
		}
	}
	if !p.HasUnresolved() {
		t.Fatal("missing no-free-output diagnostic")
	}
}

func TestNormalRetainsPolicyOutputOwnershipWithoutSelectingIt(t *testing.T) {
	c, s := voiceFixture(t)
	c.Profiles = &config.Profiles{Microphones: []string{"lav", "webcam"}}
	p := config.ProfilePolicy{Devices: config.VR{Input: 4}, Microphones: []string{"normal"}, Playback: []config.Candidate{{Driver: "wdm", Pattern: "^VR-only output$"}}, Choices: config.ProfileChoices{Source: "auto", Mode: "direct", Monitor: "off"}}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	c.PolicyPlayback = p.Playback
	s.Assignments["A2"] = "VR-only output"
	s.Devices = append(s.Devices, model.Device{Name: "VR-only output", Direction: "output", Driver: "wdm", Available: true})
	resolved := ProfileConfig(c, s)
	if resolved.ProfilePlayback == nil || resolved.ProfilePlayback.Name == "VR-only output" {
		t.Fatal("ownership changed Normal priority")
	}
	plan, err := Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Topology.PlaybackTarget != "A2" {
		t.Fatal("forgot retired output ownership", plan.Topology.PlaybackTarget)
	}
	c.Intent = c.VoiceIntent()
	c.Intent.PlaybackDevice = "unmanaged output"
	s.Devices = append(s.Devices, model.Device{Name: "unmanaged output", Direction: "output", Driver: "wdm", Available: true})
	if ProfileConfig(c, s).ProfilePlayback.Name == "unmanaged output" {
		t.Fatal("accepted unowned override")
	}
}
