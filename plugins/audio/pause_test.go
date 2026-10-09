package audio

import (
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/routing"
	"sound-snoofer/snoofer"
)

func TestPauseToggles(t *testing.T) {
	s := control.State{
		Connected: true,
		Intent:    &config.Intent{Source: "desk", Enabled: true, PauseSends: true},
		Plan:      &routing.Plan{Topology: &routing.Topology{HeldDevices: 1, HeldSends: 3}},
	}
	got := map[string]snoofer.Control{}
	for _, c := range controls(s) {
		got[c.ID] = c
	}
	devices, sends := got["audio.pause-devices"], got["audio.pause-sends"]
	if devices.Label != "Disable Device Manipulation" || devices.Value != "Off" || devices.Status != "" {
		t.Fatal(devices)
	}
	if sends.Label != "Disable Send Manipulation" || sends.Value != "On" || sends.Status != "3 held" || sends.ShortLabel != "Pause Sends" {
		t.Fatal(sends)
	}
	a, err := action(s, snoofer.Request{ID: "audio.pause-devices", Operation: "press"})
	if err != nil || a.Row != "pause-devices" || a.Value != "true" {
		t.Fatal(a, err)
	}
	a, err = action(s, snoofer.Request{ID: "audio.pause-sends", Operation: "press"})
	if err != nil || a.Row != "pause-sends" || a.Value != "false" {
		t.Fatal(a, err)
	}
}
