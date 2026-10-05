package app

import (
	"fmt"
	"strings"
	"testing"

	"sound-snoofer/snoofer"

	tea "charm.land/bubbletea/v2"
)

func TestSectionNavigationAndSnapshots(t *testing.T) {
	s := styledFixture()
	actions := make(chan UIAction, 8)
	s.actions = actions
	key := func(code rune) {
		t.Helper()
		model, _ := s.Update(tea.KeyPressMsg{Code: code})
		s = model.(screen)
	}
	key(']')
	if s.currentGroup() != "Normal playback" || len(s.rows()) != 1 {
		t.Fatal("section did not change", s.currentGroup(), s.rows())
	}
	key(']')
	if s.currentGroup() != "Shared audio" {
		t.Fatal(s.currentGroup())
	}
	key(tea.KeyDown)
	selectedID := s.rows()[s.selected].ID
	state := s.state
	state.Controls = append([]snoofer.Control(nil), state.Controls...)
	state.Controls[3], state.Controls[4] = state.Controls[4], state.Controls[3]
	model, _ := s.Update(stateMsg(state))
	s = model.(screen)
	if s.rows()[s.selected].ID != selectedID {
		t.Fatal("selection drifted")
	}
	key('+')
	action := <-actions
	if action.Request == nil || action.Request.ID != selectedID || action.Request.Operation != "adjust" {
		t.Fatal("wrong control dispatched", action)
	}
	state.Controls = state.Controls[:3]
	model, _ = s.Update(stateMsg(state))
	s = model.(screen)
	if s.currentGroup() != "Normal microphone" || s.selected != 0 {
		t.Fatal("removed group not reset")
	}
	key('[')
	if s.currentGroup() != "Normal playback" {
		t.Fatal("wrap failed")
	}
	if len(s.actions) != 0 {
		t.Fatal("browsing dispatched an action")
	}
	state.Controls = nil
	model, _ = s.Update(stateMsg(state))
	s = model.(screen)
	key(']')
	key(tea.KeyEnter)
	if len(s.rows()) != 0 || len(s.actions) != 0 {
		t.Fatal("empty state dispatched")
	}
}

func TestSectionPresentation(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	s := styledFixture()
	if strings.Contains(row("Section", "", 23, false, false), "…") {
		t.Fatal("short label spuriously truncated")
	}
	s.group = "Shared audio"
	if !strings.Contains(s.View().Content, "● On") || strings.Contains(s.View().Content, "Lavalier") {
		t.Fatal("form does not isolate selected section")
	}
	s.group = "Normal microphone"
	if strings.Count(s.View().Content, "VR overriding") != 1 {
		t.Fatal("diagnostic missing or duplicated")
	}
	// The rail scrolls independently when third-party plugins add many groups.
	for i := 0; i < 40; i++ {
		s.state.Controls = append(s.state.Controls, snoofer.Control{ID: fmt.Sprint(i), Group: fmt.Sprintf("Section %d", i), Label: "Choice", Available: true})
	}
	s.group = "Section 39"
	s.height = 10
	view := s.View().Content
	if !strings.Contains(view, "Section 39") || !strings.Contains(view, "Choice") {
		t.Fatal("last section hidden")
	}
	s.width = 60
	if !strings.Contains(s.View().Content, "44/44") {
		t.Fatal("narrow section position missing")
	}
}
