package controller

import (
	"testing"

	"sound-snoofer/internal/routing"
)

func TestPlaybackGainBusNames(t *testing.T) {
	for target, want := range map[string]string{
		"A1": "Bus[0].Gain", "A2": "Bus[1].Gain", "A5": "Bus[4].Gain",
		"": "", "A": "", "A0": "", "A6": "", "B1": "", "a1": "",
		"A01": "", "A+1": "", "A-1": "", "A 1": "", "A1 ": "", "A1x": "",
	} {
		plan := &routing.Plan{Topology: &routing.Topology{PlaybackTarget: target}}
		if got := GainTarget(plan, "playback"); got != want {
			t.Errorf("GainTarget(%q) = %q; want %q", target, got, want)
		}
	}
}
