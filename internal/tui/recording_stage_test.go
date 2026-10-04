package tui

import (
	"testing"

	"sound-snoofer/internal/config"
)

func TestDirectStageEditAndKeyboard(t *testing.T) {
	i := &config.Intent{Mode: "element", Recording: &config.RecordingChoices{MicEnabled: true, ComputerEnabled: true, MicTap: "post"}}
	if err := editIntent(i, Action{Row: "mode", Value: "direct"}); err != nil {
		t.Fatal(err)
	}
	if i.Recording.MicTap != "post" || !i.Recording.MicEnabled || !i.Recording.ComputerEnabled {
		t.Fatal(i)
	}
	if err := editIntent(i, Action{Row: "record-tap", Value: "post"}); err != nil {
		t.Fatal(err)
	}
	if i.Recording.MicTap != "post" {
		t.Fatal("invalid command accepted")
	}
	actions := make(chan Action, 1)
	s := screen{actions: actions, state: State{Intent: i}}
	for n, row := range s.rules() {
		if row.key == "record-tap" {
			s.selected = n
		}
	}

	s.ruleAction("enter")
	a := <-actions
	if a.Value != "pre" {
		t.Fatal(a)
	}
	acknowledgeEdit(t, &s, a)
	s.ruleAction("enter")
	a = <-actions
	if a.Value != "post" {
		t.Fatal("Post preference unavailable in Direct", a)
	}
}
