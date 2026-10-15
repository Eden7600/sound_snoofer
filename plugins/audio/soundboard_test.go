package audio

import (
	"fmt"
	"testing"
	"time"

	"sound-snoofer/internal/control"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

func TestSoundboardRequiresObservedRoutes(t *testing.T) {
	routes := &SoundboardRoutes{Microphone: true, Monitor: true}
	// A2 is Playback; the output slot on A4 also carries the soundboard.
	planned := map[string]int{"Strip[7].A1": 0, "Strip[7].A2": 1, "Strip[7].A3": 0, "Strip[7].A4": 1, "Strip[7].A5": 0, "Strip[7].B3": 1}
	var ops []routing.Operation
	numbers := map[string]float32{"Strip[7].B2": 0}
	for bus := 1; bus <= 5; bus++ {
		param := fmt.Sprintf("Strip[7].A%d", bus)
		ops = append(ops, routing.Operation{Parameter: param, Value: planned[param]})
		numbers[param] = float32(planned[param])
	}
	ops = append(ops, routing.Operation{Parameter: "Strip[7].B3", Value: 1})
	numbers["Strip[7].B3"] = 1
	s := control.State{Live: true, Connected: true, ObservedAt: time.Now(), Snapshot: model.Snapshot{Edition: 3, Numbers: numbers}, Plan: &routing.Plan{Topology: &routing.Topology{PlaybackTarget: "A2", Operations: ops}}}
	if err := soundboardReady(s, routes); err != nil {
		t.Fatal(err)
	}
	s.Snapshot.Numbers["Strip[7].A4"] = 0
	if soundboardReady(s, routes) == nil {
		t.Fatal("unapplied output slot accepted")
	}
	s.Snapshot.Numbers["Strip[7].A4"] = 1
	s.Snapshot.Numbers["Strip[7].B3"] = 0
	if soundboardReady(s, routes) == nil {
		t.Fatal("unapplied route accepted")
	}
	s.Snapshot.Numbers["Strip[7].B3"] = 1
	s.ObservedAt = time.Now().Add(-4 * time.Second)
	if soundboardReady(s, routes) == nil {
		t.Fatal("stale observation accepted")
	}
	s.ObservedAt = time.Now()
	s.Plan.Topology.Operations = nil
	if soundboardReady(s, routes) == nil {
		t.Fatal("unplanned routing accepted")
	}
}
