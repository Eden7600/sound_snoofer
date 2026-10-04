package tui

import (
	"strings"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

func TestControlPendingAndOverride(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	c, err := config.Decode([]byte(ruleConfig))
	if err != nil {
		t.Fatal(err)
	}
	c.Studio.Recording = &config.Recording{}
	if err = c.Validate(); err != nil {
		t.Fatal(err)
	}
	i := c.VoiceIntent()
	i.Monitor = "post"
	i.Recording.MicTap = "post"
	s := screen{width: 100, height: 40, actions: make(chan Action, 1), state: State{Intent: i, Connected: true, Snapshot: model.Snapshot{Element: &model.ProcessStatus{Known: true}}, Plan: &routing.Plan{Topology: &routing.Topology{}}}}
	for _, key := range []string{"mode", "monitor", "record-tap"} {
		r := ruleRow{key: key, value: intentValue(i, key)}
		color, text := s.controlStatus(r)
		if color != "196" || !strings.Contains(text, "→") {
			t.Fatal(key, color, text)
		}
	}
	s.state.Snapshot.Element.Running = true
	for _, r := range s.rules() {
		color, _ := s.controlStatus(r)
		if color != "" {
			t.Fatal("satisfied control highlighted", r.key, color)
		}
	}
	s.queueEdit(Action{Kind: editRule, Row: "record-loop", Value: "true"})
	for _, r := range s.rules() {
		color, _ := s.controlStatus(r)
		if r.key == "record-loop" {
			if color != "226" {
				t.Fatal("queue not yellow")
			}
		} else if color != "" {
			t.Fatal("unrelated queue highlight", r.key)
		}
	}
	// Saved preference still needs verified readback before pending clears.
	s.state.Intent = s.choices().Clone()
	s.draft = nil
	s.inflight = 0
	s.state.Plan.Topology.Operations = []routing.Operation{{Parameter: "Recorder.mode.Loop", Value: 1, Change: true}}
	r := ruleRow{key: "record-loop", value: "true"}
	if color, _ := s.controlStatus(r); color != "226" {
		t.Fatal("unverified not yellow")
	}
	for _, other := range s.rules() {
		if other.key != "record-loop" {
			if color, _ := s.controlStatus(other); color != "" {
				t.Fatal("unrelated readback highlight", other.key)
			}
		}
	}
	s.state.Plan.Topology.Operations[0].Change = false
	if color, _ := s.controlStatus(r); color != "" {
		t.Fatal("verified did not clear")
	}
	s.state.Snapshot.Element.Running = false
	text := s.View().Content
	if !strings.Contains(text, "38;5;196") {
		t.Fatal("red foreground missing")
	}
	t.Setenv("NO_COLOR", "1")
	text = s.View().Content
	if !strings.Contains(text, "→") || strings.Contains(text, "\x1b[") {
		t.Fatal("noncolor indicator missing")
	}
}
