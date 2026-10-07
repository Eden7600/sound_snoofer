package audio

import (
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
)

func TestMicMuteFollowsPreferenceAndReadback(t *testing.T) {
	if (&Instance{}).MicMute().Known {
		t.Fatal("known without a preference")
	}
	s := echoState(0, "desk", "A2")
	s.Connected = true
	s.Intent = &config.Intent{Version: 1, Enabled: true, Source: "desk", MicMuted: true}
	s.Snapshot.Numbers = map[string]float32{"Strip[0].Mute": 0, "Strip[6].Mute": 0}
	i := echoInstance(s)
	if m := i.MicMute(); !m.Known || !m.Muted || m.Applied {
		t.Fatal("unapplied preference", m)
	}
	s.Snapshot.Numbers = map[string]float32{"Strip[0].Mute": 1, "Strip[6].Mute": 1}
	if m := echoInstance(s).MicMute(); !m.Applied {
		t.Fatal("applied preference", m)
	}
	// Mic stack off: nothing to mute, so the preference counts as applied.
	off := echoState(-1, "off", "A2")
	off.Intent = &config.Intent{Version: 1, MicMuted: true}
	if m := echoInstance(off).MicMute(); !m.Known || !m.Applied {
		t.Fatal("mic stack off", m)
	}
}

func TestSetMicMuteSendsTheMicMuteEdit(t *testing.T) {
	s := echoState(0, "desk", "A2")
	s.Revision = 7
	i := echoInstance(s)
	if err := i.SetMicMute(true, "discord"); err != nil {
		t.Fatal(err)
	}
	a := <-i.actions
	if a.Kind != control.Edit || a.Row != "mic-mute" || a.Value != "true" || a.Revision != 7 || a.Origin != "discord" {
		t.Fatalf("%+v", a)
	}
	if err := i.SetMicMute(false, "discord"); err != nil {
		t.Fatal(err)
	}
	if err := i.SetMicMute(false, "discord"); err == nil {
		t.Fatal("full queue accepted")
	}
}
