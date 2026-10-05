package controller

import (
	"context"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

func TestSoundboardGainResetNeverMutes(t *testing.T) {
	f := &delayedMixer{mixerFake: mixerFake{s: model.Snapshot{Edition: 3, Numbers: map[string]float32{"Strip[7].Gain": -17, "Strip[7].Mute": 1, "Strip[0].Gain": -8}}}, lag: 2}
	c := &Controller{Backend: f, Clock: &fakeClock{}, Config: config.Config{SoundboardReserved: true, SoundboardPolicy: func() *config.SoundboardRoutes { return &config.SoundboardRoutes{Microphone: true} }}}
	ctx := context.Background()
	if err := c.Gain(ctx, "soundboard", "Strip[7].Gain|", 127, true); err != nil {
		t.Fatal(err)
	}
	if f.s.Numbers["Strip[7].Gain"] != 12 {
		t.Fatal("gain did not clamp")
	}
	if err := c.ResetGain(ctx, "soundboard", "Strip[7].Gain|", true); err != nil {
		t.Fatal(err)
	}
	if f.s.Numbers["Strip[7].Gain"] != 0 || f.s.Numbers["Strip[7].Mute"] != 1 || f.s.Numbers["Strip[0].Gain"] != -8 {
		t.Fatal("reset changed mute or another strip")
	}
	for _, parameter := range f.writes {
		if parameter != "Strip[7].Gain" {
			t.Fatal("wrong parameter", parameter)
		}
	}
	before := len(f.writes)
	if c.ResetGain(ctx, "soundboard", "stale", true) == nil {
		t.Fatal("stale target accepted")
	}
	if c.ResetGain(ctx, "soundboard", "Strip[7].Gain|", false) == nil {
		t.Fatal("preview applied")
	}
	c.Config.SoundboardPolicy = nil
	if c.ResetGain(ctx, "soundboard", "Strip[7].Gain|", true) == nil {
		t.Fatal("disabled soundboard accepted")
	}
	if len(f.writes) != before {
		t.Fatal("rejected request wrote mixer")
	}
}
