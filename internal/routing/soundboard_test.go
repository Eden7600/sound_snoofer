package routing

import (
	"sound-snoofer/internal/config"
	"testing"
)

func TestReservedSoundboardRoutesAndTeardown(t *testing.T) {
	c, s := voiceFixture(t)
	c.SoundboardReserved = true
	c.SoundboardPolicy = func() *config.SoundboardRoutes { return &config.SoundboardRoutes{Microphone: true, Monitor: true} }
	intent := c.VoiceIntent()
	intent.Enabled = false
	c.Intent = intent
	p, err := Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	applyPlan(&s, p)
	if s.Numbers["Strip[7].B3"] != 1 || s.Numbers["Strip[7].B2"] != 0 || s.Numbers["Strip[7]."+p.Topology.PlaybackTarget] != 1 {
		t.Fatal("soundboard did not bypass disabled mic stack")
	}
	c.SoundboardPolicy = nil
	p, err = Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	applyPlan(&s, p)
	if s.Numbers["Strip[7].B3"] != 0 || s.Numbers["Strip[7]."+p.Topology.PlaybackTarget] != 0 {
		t.Fatal("disabled plugin sends retained")
	}
}
