package control

import (
	"testing"

	"sound-snoofer/internal/config"
)

func TestEditPauseRows(t *testing.T) {
	i := &config.Intent{}
	for _, a := range []Action{{Row: "pause-devices", Value: "true"}, {Row: "pause-sends", Value: "true"}} {
		if err := EditIntent(i, a); err != nil {
			t.Fatal(err)
		}
	}
	if !i.PauseDevices || !i.PauseSends {
		t.Fatal(i)
	}
	if err := EditIntent(i, Action{Row: "pause-sends", Value: "maybe"}); err == nil || !i.PauseSends {
		t.Fatal("invalid value accepted", err)
	}
}
