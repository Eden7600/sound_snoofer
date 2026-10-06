package routing

import (
	"slices"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

func TestASIOPriorityDeterminesMicrophoneAndPlaybackEligibility(t *testing.T) {
	c, s := voiceFixture(t)
	c.Studio.ASIO = append([]config.ASIOInterface{{ASIOPattern: "^Other ASIO$", PresencePattern: "^Other input$", Inputs: [2]int{0, 0}}}, c.Studio.ASIO...)
	c.Studio.Playback = append([]config.Candidate{{Driver: "asio", Pattern: "^Volt ASIO$"}}, c.Studio.Playback...)
	c.Profiles = &config.Profiles{Microphones: []string{"lav", "desk", "webcam"}}
	c.Intent = c.VoiceIntent()
	c.Intent.Source = "lav"
	c.Intent.PlaybackDevice = "Volt ASIO"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Devices = append(s.Devices, model.Device{Name: "Other ASIO", Driver: "asio", Direction: "output"}, model.Device{Name: "Other input", Driver: "wdm", Direction: "input", Available: true})
	p, err := Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if p.Topology.Voice.Effective != "webcam" || p.Topology.PlaybackTarget != "A2" {
		t.Fatalf("ineligible interface used: %+v", p.Topology)
	}
	if slices.Contains(PlaybackOptions(c, s), "Volt ASIO") || slices.Contains(MicrophoneOptions(c, s), "lav") {
		t.Fatal("unselected ASIO exposed")
	}
	for _, op := range p.Topology.Operations {
		if op.Target == "A1" && op.Device.Name != "Other ASIO" {
			t.Fatal("playback changed interface priority")
		}
		if op.Parameter == "Patch.asio[0]" && op.Value != 0 {
			t.Fatal("unavailable input patched")
		}
	}
	applyPlan(&s, p)
	s.Devices[len(s.Devices)-1].Available = false
	p, err = Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if p.Topology.Voice.Effective != "lav" || p.Topology.PlaybackTarget != "A1" {
		t.Fatal("priority fallback did not restore Volt", p.Topology)
	}
	c.Studio.ASIO = nil
	p, err = Build(c, s)
	// The former interface is now unmanaged: never overwrite it silently.
	if err == nil && p.Topology.ASIOActive {
		t.Fatal("empty priority activated ASIO")
	}
}

func TestASIOChannelMappingAndAmbiguity(t *testing.T) {
	c, s := voiceFixture(t)
	c.Studio.ASIO[0].Inputs = [2]int{4, 0}
	p, err := Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range p.Topology.Operations {
		if op.Parameter == "Patch.asio[0]" && op.Value != 4 {
			t.Fatal(op)
		}
		if op.Parameter == "Patch.asio[2]" && op.Value != 0 {
			t.Fatal(op)
		}
	}
	s.Devices = append(s.Devices, s.Devices[1])
	if _, err = Build(c, s); err == nil {
		t.Fatal("ambiguous interface accepted")
	}
}
