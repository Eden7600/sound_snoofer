package tui

import (
	"testing"

	"voice-snooter/internal/config"
)

func TestDirectStageEditAndKeyboard(t *testing.T) {
	i := &config.Intent{Mode: "element", Recording: &config.RecordingChoices{MicEnabled: true, ComputerEnabled: true, MicTap: "post"}}
	if err := editIntent(i, Action{Row: "mode", Value: "direct"}); err != nil {
		t.Fatal(err)
	}
	if i.Recording.MicTap != "pre" || !i.Recording.MicEnabled || !i.Recording.ComputerEnabled {
		t.Fatal(i)
	}
	if err := editIntent(i, Action{Row: "record-tap", Value: "post"}); err != nil {
		t.Fatal(err)
	}
	if i.Recording.MicTap != "pre" {
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
	if len(actions) != 0 || s.pending != "Pre only in Direct" {
		t.Fatal("Direct allowed Post")
	}
	if err := editIntent(i, Action{Row: "mode", Value: "element"}); err != nil {
		t.Fatal(err)
	}
	s.ruleAction("enter")
	if a := <-actions; a.Value != "post" {
		t.Fatal(a)
	}
}
