package audio

import (
	"maps"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
)

func TestSelectionLabelsBelongToTheirControl(t *testing.T) {
	s, _ := guiState(t, true)
	s.VRConfigured = true
	s.Intent.VRProfile = &config.ProfileChoices{Source: "auto", Mode: "element", Monitor: "post", Playback: "lav"}
	s.ChoiceLabels = map[string]string{"desk": "Desk microphone", "lav": "lAv"}
	s.SilentMics = []string{"lav"}
	s.OutputOptions, s.VROutputOptions = []string{"", "lav"}, []string{"", "lav"}
	s.Intent.PlaybackDevice = "lav"
	s.VRSourceOptions = []string{"desk", "lav", "off"}
	published := controls(s)
	for _, group := range []struct {
		ids    []string
		labels map[string]string
	}{
		{[]string{"source", "normal-source", "vr-profile-source"}, map[string]string{"auto": "Automatic", "off": "Off", "desk": "Desk microphone", "lav": "lAv · Silent"}},
		{[]string{"mode", "normal-mode", "vr-profile-mode"}, map[string]string{"direct": "Direct", "element": "Element"}},
		{[]string{"monitor", "normal-monitor", "vr-profile-monitor"}, map[string]string{"off": "Off", "pre": "Pre", "post": "Post"}},
		{[]string{"record-tap"}, map[string]string{"pre": "Pre", "post": "Post"}},
		{[]string{"output", "normal-output", "vr-profile-output"}, map[string]string{"": "Automatic"}},
	} {
		for _, id := range group.ids {
			t.Run(id, func(t *testing.T) {
				c := controlByID(t, published, "audio."+id)
				if !maps.Equal(c.OptionLabels, group.labels) {
					t.Fatalf("labels = %v, want %v", c.OptionLabels, group.labels)
				}
			})
		}
	}
	for _, c := range published {
		if c.Kind != "selection" && len(c.OptionLabels) != 0 {
			t.Errorf("%s labels a nonselection: %v", c.ID, c.OptionLabels)
		}
	}
	if got := controlByID(t, published, "audio.output").Value; got != "lav" {
		t.Fatalf("device name changed: %q", got)
	}
}

func TestCompactLabelsAndStateIcons(t *testing.T) {
	s := control.State{Connected: true, Profile: "Normal", Intent: &config.Intent{Enabled: false, Source: "desk", Mode: "direct", Monitor: "off", VRProfile: &config.ProfileChoices{Source: "auto", Mode: "element", Monitor: "pre"}, Recording: &config.RecordingChoices{MicTap: "pre"}}}
	for _, mode := range []string{"direct", "element"} {
		for _, tap := range []string{"pre", "post"} {
			s.Intent.Mode = mode
			s.Intent.Recording.MicTap = tap
			for _, c := range controls(s) {
				if len(c.ShortLabel) > 16 {
					t.Fatal("label will clip", c.ID, c.ShortLabel)
				}
				switch c.ID {
				case "audio.mic-mute":
					if c.ShortLabel != "Mute" || c.Label != "Mic Mute" {
						t.Fatal(c)
					}
				case "audio.monitor":
					if c.ShortLabel != "Monitor" {
						t.Fatal(c)
					}
				case "audio.mode":
					if c.ShortLabel != "Mic Path" || c.Icon != "mode-"+mode {
						t.Fatal(c)
					}
				case "audio.record-tap":
					if c.Icon != "tap-"+tap {
						t.Fatal(c)
					}
				case "audio.mic-stack":
					if c.ShortLabel != "Mic Stack" || c.Status != "" {
						t.Fatal(c)
					}
				}
			}
		}
	}
}
