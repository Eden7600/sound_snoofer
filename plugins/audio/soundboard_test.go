package audio

import (
	"testing"
	"time"

	"sound-snoofer/internal/control"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

func TestSoundboardRequiresObservedRoutes(t *testing.T) {
	routes := &SoundboardRoutes{Microphone: true, Monitor: true}
	s := control.State{Live: true, Connected: true, ObservedAt: time.Now(), Snapshot: model.Snapshot{Edition: 3, Numbers: map[string]float32{"Strip[7].B2": 0, "Strip[7].B3": 1, "Strip[7].A1": 0, "Strip[7].A2": 1, "Strip[7].A3": 0, "Strip[7].A4": 0, "Strip[7].A5": 0}}, Plan: &routing.Plan{Topology: &routing.Topology{PlaybackTarget: "A2"}}}
	if err := soundboardReady(s, routes); err != nil {
		t.Fatal(err)
	}
	s.Snapshot.Numbers["Strip[7].B3"] = 0
	if soundboardReady(s, routes) == nil {
		t.Fatal("unapplied route accepted")
	}
	s.Snapshot.Numbers["Strip[7].B3"] = 1
	s.ObservedAt = time.Now().Add(-4 * time.Second)
	if soundboardReady(s, routes) == nil {
		t.Fatal("stale observation accepted")
	}
}
