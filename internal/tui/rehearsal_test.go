package tui

import (
	"strings"
	"testing"

	"sound-snoofer/internal/config"
)

func TestRehearsalControls(t *testing.T) {
	c, err := config.Decode([]byte(ruleConfig))
	if err != nil {
		t.Fatal(err)
	}
	c.Studio.Recording = &config.Recording{}
	if err = c.Validate(); err != nil {
		t.Fatal(err)
	}
	actions := make(chan Action, 1)
	s := screen{width: 140, height: 40, state: State{Intent: c.VoiceIntent()}, actions: actions}
	for _, stage := range []string{"pre", "post"} {
		s.state.Intent.Monitor = stage
		label := "Pre-VST"
		if stage == "post" {
			label = "Post-VST"
		}
		if !strings.Contains(s.View().Content, label) {
			t.Fatal("missing label", label)
		}
	}
	for n, row := range s.rules() {
		s.selected = n
		switch row.key {
		case "record-vst", "record-loop":
			s.ruleAction("enter")
			a := <-actions
			if a.Kind != editRule || a.Value != "true" {
				t.Fatal(a)
			}
			acknowledgeEdit(t, &s, a)
		case "snippet-play", "snippet-stop":
			s.ruleAction("enter")
			a := <-actions
			want := playSnippet
			if row.key == "snippet-stop" {
				want = stopRecording
			}
			if a.Kind != want {
				t.Fatal(a)
			}
		}
	}
	if !s.choices().Recording.ToVST || !s.choices().Recording.Loop {
		t.Fatal("choices not saved")
	}
	if err := editIntent(s.state.Intent, Action{Row: "source", Value: "off"}); err != nil {
		t.Fatal(err)
	}
	if s.state.Intent.Recording.ToVST {
		t.Fatal("Off retained rehearsal")
	}
}
