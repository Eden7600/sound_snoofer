package audio

import (
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
)

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
					if c.ShortLabel != "Mute" || c.Label != "Mic mute" {
						t.Fatal(c)
					}
				case "audio.monitor":
					if c.ShortLabel != "Monitor" {
						t.Fatal(c)
					}
				case "audio.mode":
					if c.ShortLabel != "Mic processing" || c.Icon != "mode-"+mode {
						t.Fatal(c)
					}
				case "audio.record-tap":
					if c.Icon != "tap-"+tap {
						t.Fatal(c)
					}
				case "audio.mic-stack":
					if c.ShortLabel != "Mic stack" || c.Status != "" {
						t.Fatal(c)
					}
				}
			}
		}
	}
}
