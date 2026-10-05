package control

import (
	"sound-snoofer/internal/config"
	"testing"
)

func TestStackTargetEditsNeverEnable(t *testing.T) {
	i := &config.Intent{Source: "lav", Enabled: false, Mode: "direct", Monitor: "off", MicMuted: true, VRProfile: &config.ProfileChoices{Source: "auto", Mode: "direct", Monitor: "off"}}
	for _, a := range []Action{{Row: "normal-source", Value: "webcam"}, {Row: "vr-profile-source", Value: "vr:headset"}, {Row: "source", Value: "auto"}} {
		if err := EditIntent(i, a); err != nil {
			t.Fatal(err)
		}
		if i.Enabled || !i.MicMuted {
			t.Fatal("target changed master or mute")
		}
	}
	for _, row := range []string{"source", "normal-source", "vr-profile-source"} {
		if EditIntent(i, Action{Row: row, Value: "off"}) == nil {
			t.Fatal("Off remains a target")
		}
	}
	for _, enabled := range []string{"true", "false"} {
		if err := EditIntent(i, Action{Row: "mic-stack", Value: enabled}); err != nil {
			t.Fatal(err)
		}
		if i.Source != "auto" || i.VRProfile.Source != "vr:headset" || !i.MicMuted {
			t.Fatal("master lost target/mute")
		}
	}
}
