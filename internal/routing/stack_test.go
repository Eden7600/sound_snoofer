package routing

import (
	"fmt"
	"strings"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

func TestStackMasterSurvivesProfilesAndRestoresTarget(t *testing.T) {
	c, s := recordingFixture(t)
	c.Profiles = &config.Profiles{Microphones: []string{"lav", "webcam"}}
	c.Intent = c.VoiceIntent()
	c.Intent.Source = "webcam"
	c.Intent.MicMuted = true
	c.Intent.Recording.ComputerEnabled = true
	p := config.ProfilePolicy{Devices: config.VR{Input: 4, Headsets: []config.Headset{{ID: "headset", Microphone: "^Headset mic$"}}}, Microphones: []string{"vr:headset"}, Playback: []config.Candidate{{Driver: "normal"}}, Choices: config.ProfileChoices{Source: "auto", Mode: "element", Monitor: "post"}, Known: true}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Policy = func() *config.ProfilePolicy { return &p }
	s.Devices = append(s.Devices, model.Device{Name: "Headset mic", Driver: "wdm", Direction: "input", Available: true})
	s.Recorder.Values["Recorder.stop"] = 0
	s.Recorder.Values["Recorder.record"] = 1
	for _, running := range []bool{false, true, false, true} {
		p.Running = running
		c.Intent.Enabled = false
		// Simulate an existing route before disabling, including the processing return.
		s.Assignments["input:4"] = "Headset mic"
		for _, strip := range []int{0, 1, 2, 3, 6} {
			for _, bus := range []string{"A1", "A2", "A3", "A4", "A5", "B1", "B2", "B3"} {
				s.Numbers[fmt.Sprintf("Strip[%d].%s", strip, bus)] = 1
			}
		}
		plan, err := Build(c, s)
		if err != nil {
			t.Fatal(err)
		}
		if plan.Topology.Voice.Effective != "off" {
			t.Fatal("profile re-enabled master", running, plan.Topology.Voice)
		}
		for _, op := range plan.Topology.Operations {
			if strings.HasPrefix(op.Parameter, "Recorder.") && (op.Parameter == "Recorder.stop" || op.Parameter == "Recorder.record" || op.Parameter == "Recorder.play") {
				t.Fatal("transport changed", op)
			}
		}
		applyPlan(&s, plan)
		for _, slot := range []string{"input:1", "input:2", "input:3", "input:4"} {
			if s.Assignments[slot] != "" {
				t.Fatal("input retained", slot)
			}
		}
		for n := 0; n < 4; n++ {
			if s.Numbers[fmt.Sprintf("Patch.asio[%d]", n)] != 0 {
				t.Fatal("patch retained")
			}
		}
		for _, strip := range []int{0, 1, 2, 3, 6} {
			for _, bus := range []string{"A1", "A2", "A3", "A4", "A5", "B1", "B2", "B3"} {
				if s.Numbers[fmt.Sprintf("Strip[%d].%s", strip, bus)] != 0 {
					t.Fatal("mic send retained", strip, bus)
				}
			}
		}
		if s.Assignments["A1"] != "Volt ASIO" || s.Assignments["A2"] != "speakers" || s.Numbers["Strip[5].A2"] != 1 || s.Recorder.State() != "Recording" {
			t.Fatal("playback or recording changed")
		}
		c.Intent.Source = "lav" // Editing inactive/disabled targets must not reconnect.
		plan, err = Build(c, s)
		if err != nil || plan.Topology.Voice.Effective != "off" {
			t.Fatal(err)
		}
		c.Intent.Enabled = true
		plan, err = Build(c, s)
		if err != nil {
			t.Fatal(err)
		}
		want := "lav"
		if running {
			want = "vr:headset"
		}
		if plan.Topology.Voice.Effective != want || !c.VoiceIntent().MicMuted {
			t.Fatal("restore lost target or mute", plan.Topology.Voice)
		}
	}
}
