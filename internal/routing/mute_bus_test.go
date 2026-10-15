package routing

import (
	"testing"

	"sound-snoofer/internal/config"
)

func TestPlaybackMuteBusNames(t *testing.T) {
	for target, want := range map[string]string{
		"A1": "Bus[0].Mute", "A2": "Bus[1].Mute", "A5": "Bus[4].Mute",
		"": "", "A": "", "A0": "", "A6": "", "B1": "", "a1": "",
		"A01": "", "A+1": "", "A-1": "", "A 1": "", "A1 ": "", "A1x": "",
	} {
		plan := &Plan{Topology: &Topology{PlaybackTarget: target}}
		got, requested, handled := MuteTargets(&config.Intent{PlaybackMuted: true}, plan, "speaker-mute")
		if !requested || !handled || (want == "" && len(got) != 0) || (want != "" && (len(got) != 1 || got[0] != want)) {
			t.Errorf("MuteTargets(%q) = %v, %v, %v; want %q, true, true", target, got, requested, handled, want)
		}
	}
}
