package control

import (
	"slices"
	"testing"
	"time"

	"sound-snoofer/internal/config"
)

func TestActivityLatch(t *testing.T) {
	a := &config.Activity{Check: 2, SilenceDB: -70, SilentAfterS: 10}
	quiet, loud := float32(0.0001), float32(0.1) // -80 and -20 dBFS.
	var l activityLatch
	start := time.Unix(0, 0)
	at := func(d time.Duration) time.Time { return start.Add(d) }
	if l.observe(a, map[string]float32{"lav": quiet, "desk": loud}, at(0)) {
		t.Fatal("silence latched immediately")
	}
	if l.observe(a, map[string]float32{"lav": quiet, "desk": loud}, at(9*time.Second)) {
		t.Fatal("silence latched early")
	}
	if !l.observe(a, map[string]float32{"lav": quiet, "desk": loud}, at(10*time.Second)) || !slices.Equal(l.silent(), []string{"lav"}) {
		t.Fatal("silence not latched", l.silent())
	}
	// A brief blip does not unlatch; 300 ms of signal does.
	l.observe(a, map[string]float32{"lav": loud}, at(11*time.Second))
	if l.observe(a, map[string]float32{"lav": loud}, at(11*time.Second+200*time.Millisecond)) {
		t.Fatal("unlatched before 300 ms")
	}
	if !l.observe(a, map[string]float32{"lav": loud}, at(11*time.Second+300*time.Millisecond)) || len(l.silent()) != 0 {
		t.Fatal("signal did not unlatch")
	}
	// Signal interrupts a silence run, which starts over.
	l.observe(a, map[string]float32{"lav": quiet}, at(20*time.Second))
	l.observe(a, map[string]float32{"lav": loud}, at(25*time.Second))
	l.observe(a, map[string]float32{"lav": quiet}, at(26*time.Second))
	if l.observe(a, map[string]float32{"lav": quiet}, at(31*time.Second)); len(l.silent()) != 0 {
		t.Fatal("interrupted silence latched")
	}
	// An unknown reading is never silent and resets the run.
	l.observe(a, map[string]float32{"lav": quiet}, at(40*time.Second))
	l.observe(a, map[string]float32{}, at(45*time.Second))
	if l.observe(a, map[string]float32{"lav": quiet}, at(52*time.Second)); len(l.silent()) != 0 {
		t.Fatal("unknown reading counted toward silence")
	}
	if dbfs(0) > -1000 || dbfs(1) != 0 {
		t.Fatal("dbfs")
	}
}

func TestReadyMicrophonesAreNotMetered(t *testing.T) {
	c := config.Config{Studio: &config.Studio{Microphones: []config.Microphone{{ID: "desk", Name: "Desk"}, {ID: "lav", Name: "Lav", Ready: true}}}}
	if got := meteredMics(c, []string{"lav", "desk", "gone"}); !slices.Equal(got, []string{"desk"}) {
		t.Fatal(got)
	}
}
