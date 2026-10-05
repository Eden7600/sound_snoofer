package audio

import (
	"math"
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/routing"
	"testing"
	"time"
)

func TestGainMeterFollowsMicSource(t *testing.T) {
	s := control.State{Connected: true, Intent: &config.Intent{Enabled: true, Source: "webcam"},
		LevelsAt: time.Now(), Levels: map[string]float32{"Bus[0].Gain": 0.5, "Strip[2].Gain": 0.1},
		Plan: &routing.Plan{Topology: &routing.Topology{Voice: &routing.VoiceStatus{Strip: 2}}}}
	check := func(micKnown bool) {
		t.Helper()
		found := false
		for _, c := range controls(s) {
			if c.ID == "audio.gain-mic" {
				found = true
				if c.Meter.Known != micKnown || !c.Meter.Present {
					t.Fatal(c.Meter)
				}
				if micKnown && (c.Meter.DB < -20.01 || c.Meter.DB > -19.99) {
					t.Fatal(c.Meter)
				}
			}
		}
		if !found {
			t.Fatal("mic control missing")
		}
	}
	check(true)
	for _, bad := range []float32{-1, float32(math.NaN()), float32(math.Inf(1))} {
		s.Levels["Strip[2].Gain"] = bad
		check(false)
		for _, c := range controls(s) {
			if math.IsNaN(c.Meter.DB) || math.IsInf(c.Meter.DB, 0) {
				t.Fatal("nonfinite telemetry breaks IPC", c)
			}
		}
	}
	s.Levels["Strip[2].Gain"] = 0.1
	s.Plan.Topology.Voice.Strip = 3
	check(false)
	s.Plan.Topology.Voice.Strip = -1
	check(false)
	s.Plan.Topology.Voice.Strip = 2
	s.Connected = false
	check(false)
}
