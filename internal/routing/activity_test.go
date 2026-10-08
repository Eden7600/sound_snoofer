package routing

import (
	"fmt"
	"slices"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

// activityFixture: Auto over lav, desk, webcam with activity metering.
func activityFixture(t *testing.T, check int) (config.Config, model.Snapshot) {
	t.Helper()
	c, s := voiceFixture(t)
	c.Profiles = &config.Profiles{Microphones: []string{"lav", "desk", "webcam"}}
	if check > 0 {
		c.Profiles.Activity = &config.Activity{Check: check}
	}
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	c.Intent = c.VoiceIntent()
	c.Intent.Source = "auto"
	return c, s
}

func patches(s model.Snapshot) []float32 {
	out := []float32{}
	for n := 0; n < 4; n++ {
		out = append(out, s.Numbers[fmt.Sprintf("Patch.asio[%d]", n)])
	}
	return out
}

func TestActivityWiresCheckedMicrophones(t *testing.T) {
	for _, tc := range []struct {
		check   int
		patches []float32
		webcam  string
	}{
		{0, []float32{1, 1, 2, 2}, "webcam"}, // Without activity everything is wired.
		{2, []float32{1, 1, 2, 2}, ""},
		{1, []float32{0, 0, 2, 2}, ""},
		{3, []float32{1, 1, 2, 2}, "webcam"},
	} {
		c, s := activityFixture(t, tc.check)
		p := converge(t, c, &s)
		if p.Topology.Voice.Effective != "lav" || !slices.Equal(patches(s), tc.patches) || s.Assignments["input:3"] != tc.webcam {
			t.Fatal(tc.check, p.Topology.Voice.Effective, patches(s), s.Assignments["input:3"])
		}
	}
}

func TestAutoSkipsSilentMicrophones(t *testing.T) {
	c, s := activityFixture(t, 1)
	c.SilentMics = []string{"lav"}
	p := converge(t, c, &s)
	// The checked lav stays wired so its return is noticed; desk is in use.
	if p.Topology.Voice.Effective != "desk" || !slices.Equal(patches(s), []float32{1, 1, 2, 2}) {
		t.Fatal(p.Topology.Voice.Effective, patches(s))
	}
	// The lav returns.
	c.SilentMics = nil
	if p = converge(t, c, &s); p.Topology.Voice.Effective != "lav" || !slices.Equal(patches(s), []float32{0, 0, 2, 2}) {
		t.Fatal(p.Topology.Voice.Effective, patches(s))
	}
	// Silence never selects Off: with every option silent, priority applies.
	c.SilentMics = []string{"lav", "desk", "webcam"}
	if p = converge(t, c, &s); p.Topology.Voice.Effective != "lav" {
		t.Fatal(p.Topology.Voice.Effective)
	}
}

func TestSilenceIgnoredWithoutActivityOrForExplicitChoice(t *testing.T) {
	c, s := activityFixture(t, 0)
	c.SilentMics = []string{"lav"}
	if p := converge(t, c, &s); p.Topology.Voice.Effective != "lav" {
		t.Fatal("silence used without activity", p.Topology.Voice.Effective)
	}
	c, s = activityFixture(t, 2)
	c.Intent.Source = "lav"
	c.SilentMics = []string{"lav"}
	if p := converge(t, c, &s); p.Topology.Voice.Effective != "lav" {
		t.Fatal("explicit choice replaced", p.Topology.Voice.Effective)
	}
}

func TestActivityWiresFallbackWebcam(t *testing.T) {
	c, s := activityFixture(t, 1)
	for n := range s.Devices {
		if s.Devices[n].Name == "Volt input" {
			s.Devices[n].Available = false
		}
	}
	p := converge(t, c, &s)
	if p.Topology.Voice.Effective != "webcam" || s.Assignments["input:3"] != "webcam" {
		t.Fatal(p.Topology.Voice.Effective, s.Assignments["input:3"])
	}
}

func TestActivityMicStackDisabled(t *testing.T) {
	c, s := activityFixture(t, 3)
	c.Intent.Enabled = false
	converge(t, c, &s)
	if !slices.Equal(patches(s), []float32{0, 0, 0, 0}) || s.Assignments["input:3"] != "" {
		t.Fatal(patches(s), s.Assignments["input:3"])
	}
}
