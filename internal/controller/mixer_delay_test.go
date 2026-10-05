package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

type delayedMixer struct {
	mixerFake
	pending    string
	value      float32
	reads, lag int
}

func (f *delayedMixer) SetMixer(p string, v float32) error {
	f.writes = append(f.writes, p)
	if f.fail {
		return errors.New("setter failed")
	}
	f.pending, f.value, f.reads = p, v, 0
	return nil
}
func (f *delayedMixer) Snapshot() (model.Snapshot, error) {
	if f.pending != "" {
		f.reads++
		if f.reads > f.lag {
			f.s.Numbers[f.pending] = f.value
			f.pending = ""
		}
	}
	return f.s, nil
}
func TestGainDelayedVerification(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		lag                     int
		fail, cancel, wantError bool
	}{
		{name: "propagates", lag: 3}, {name: "timeout", lag: 100, wantError: true}, {name: "setter failure", fail: true, wantError: true}, {name: "cancel", lag: 100, cancel: true, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &delayedMixer{mixerFake: mixerFake{s: model.Snapshot{Edition: 3, Numbers: map[string]float32{"Bus[1].Gain": -10}, Assignments: map[string]string{"A2": "speakers"}}, fail: tc.fail}, lag: tc.lag}
			clock := &fakeClock{}
			c := &Controller{Backend: f, Clock: clock}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				clock.onWait = cancel
			}
			err := c.Gain(ctx, "A2", "Bus[1].Gain|speakers", 1, true)
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v", err)
			}
			if len(f.writes) != 1 {
				t.Fatal("relative write repeated", f.writes)
			}
			if clock.Now().Sub(time.Time{}) > 100*time.Millisecond {
				t.Fatal("verification exceeded bound")
			}
			if !tc.wantError {
				if err := c.Gain(ctx, "A2", "Bus[1].Gain|speakers", 1, true); err != nil {
					t.Fatal(err)
				}
				if f.s.Numbers["Bus[1].Gain"] != -8 {
					t.Fatal("relative turns lost")
				}
			}
		})
	}
}
func TestDelayedMuteBlocksRoutesWithoutErrorEvent(t *testing.T) {
	f := &delayedMixer{mixerFake: mixerFake{s: model.Snapshot{Edition: 3, Numbers: map[string]float32{"Bus[1].Mute": 0}}}, lag: 3}
	cfg := config.Config{Studio: &config.Studio{Voice: &config.Voice{}}, Intent: &config.Intent{Version: 1, Source: "off", Mode: "direct", Monitor: "off", BusMuted: [2]bool{false, true}}}
	mixer := &Mixer{}
	err := mixer.Reconcile(f, cfg, routing.Plan{}, f.s, false)
	if !errors.Is(err, ErrMixerPending) {
		t.Fatalf("expected pending, got %v", err)
	}
	if _, owned := mixer.Owned["Bus[1].Mute"]; !owned {
		t.Fatal("pending mute lost ownership")
	}
	f.s.Assignments = map[string]string{}
	for _, slot := range model.Slots(3) {
		f.s.Assignments[slot] = ""
	}
	for n := 0; n < 8; n++ {
		f.s.Numbers[fmt.Sprintf("Patch.asio[%d]", n)] = 0
		for bus := 1; bus <= 8; bus++ {
			f.s.Numbers[fmt.Sprintf("Strip[%d].A%d", n, bus)] = 0
			f.s.Numbers[fmt.Sprintf("Strip[%d].B%d", n, bus)] = 0
		}
	}
	// With an unresolved studio plan, the pending mute must still prevent route writes.
	c := &Controller{Backend: f, Config: cfg, Clock: &fakeClock{}, Mixer: mixer, Emit: func(e Event) {
		if e.Kind == "error" {
			t.Errorf("pending emitted failure: %s", e.Message)
		}
	}}
	if delay := c.Step(context.Background(), true); delay != 20*time.Millisecond {
		t.Fatalf("pending delay %s", delay)
	}
}
