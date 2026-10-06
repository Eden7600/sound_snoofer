package audio

import (
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
	"sound-snoofer/snoofer"
)

func TestSemanticPlaybackControlAndRequestedMute(t *testing.T) {
	s := control.State{Live: true, Connected: true, Intent: &config.Intent{PlaybackMuted: false}, LevelsAt: time.Now(), Levels: map[string]float32{"Bus[2].Gain": 0.5},
		Snapshot: model.Snapshot{Assignments: map[string]string{"A3": "Phones"}, Numbers: map[string]float32{"Bus[2].Gain": -3, "Bus[2].Mute": 1}},
		Plan:     &routing.Plan{Topology: &routing.Topology{PlaybackTarget: "A3"}}}
	var gain snoofer.Control
	for _, c := range controls(s) {
		switch c.ID {
		case "audio.a1-mute", "audio.a2-mute", "audio.gain-A1", "audio.gain-A2":
			t.Fatal("physical bus control leaked", c.ID)
		case "audio.gain-playback":
			gain = c
		case "audio.speaker-mute":
			if c.Value != "Off" || c.Status != "Pending" {
				t.Fatal("native mute replaced requested state", c)
			}
		}
	}
	if !gain.Available || !gain.Meter.Known || gain.ShortLabel != "Playback" || gain.Value != "-3.0 dB" {
		t.Fatal(gain)
	}
	a, err := action(s, snoofer.Request{ID: gain.ID, Operation: "press"})
	if err != nil || a.Row != "speaker-mute" || a.Value != "true" {
		t.Fatal(a, err)
	}
	s.Plan.Topology.PlaybackTarget = ""
	for _, c := range controls(s) {
		if c.ID == gain.ID && (c.Available || c.Meter.Known) {
			t.Fatal("unavailable playback actionable", c)
		}
	}
}
