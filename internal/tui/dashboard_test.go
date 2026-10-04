package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"sound-snoofer/internal/config"
)

func TestSourceOffKeyboardAndRestore(t *testing.T) {
	c, _ := config.Decode([]byte(ruleConfig))
	actions := make(chan Action, 8)
	s := screen{state: State{MicOptions: []string{"desk", "lav", "webcam", "off"}, Intent: c.VoiceIntent(), Revision: 2}, actions: actions}
	for _, row := range s.rules() {
		if row.key == "voice" {
			t.Fatal("duplicate switch")
		}
	}
	s.ruleAction("enter")
	s.pickerKey("down")
	s.pickerKey("down")
	s.pickerKey("down")
	s.pickerKey("enter")
	a := <-actions
	if a.Row != "source" || a.Value != "off" {
		t.Fatal(a)
	}
	i := s.state.Intent
	i.Monitor = "post"
	i.Mode = "element"
	editIntent(i, a)
	if i.MicActive() {
		t.Fatal("Off ignored")
	}
	acknowledgeEdit(t, &s, a)
	s.ruleAction(" ")
	s.pickerKey("up")
	s.pickerKey("up")
	s.pickerKey("up")
	s.pickerKey("enter")
	a = <-actions
	if a.Value != "desk" {
		t.Fatal(a)
	}
	editIntent(i, a)
	if !i.MicActive() || i.Monitor != "post" || i.Mode != "element" {
		t.Fatal("preferences lost")
	}
	acknowledgeEdit(t, &s, a)
	s.ruleAction("enter")
	s.pickerKey("down")
	s.pickerKey("enter")
	if a = <-actions; a.Value != "lav" {
		t.Fatal(a)
	}
}
func TestDashboardBoundsSelectionAndColor(t *testing.T) {
	c, _ := config.Decode([]byte(ruleConfig))
	c.Studio.Recording = &config.Recording{}
	c.Validate()
	s := screen{state: State{MicOptions: []string{"desk", "lav", "webcam", "off"}, Intent: c.VoiceIntent(), Notice: "bad\x1b[2J\nINJECT"}}
	for _, w := range []int{20, 42, 60, 100, 140} {
		for _, h := range []int{3, 10, 16, 35} {
			s.width = w
			s.height = h
			s.selected = len(s.rules()) - 1
			text := s.View().Content
			if len(strings.Split(text, "\n")) > h {
				t.Fatal("height", w, h)
			}
			for _, line := range strings.Split(text, "\n") {
				if ansi.StringWidth(line) > w-1 {
					t.Fatal("width", w, h, line)
				}
			}
			if strings.Contains(text, "\x1b[2J") || strings.Contains(text, "\nINJECT") {
				t.Fatal("unsafe text")
			}
			if w >= 42 && h >= 10 && !strings.Contains(ansi.Strip(text), "› Stop Playback") {
				t.Fatal("selection hidden", w, h, text)
			}
		}
	}
	s.width = 120
	s.height = 35
	t.Setenv("NO_COLOR", "1")
	if strings.Contains(s.View().Content, "\x1b[") {
		t.Fatal("NO_COLOR ignored")
	}
	t.Setenv("NO_COLOR", "")
	if !strings.Contains(s.View().Content, "\x1b[") {
		t.Fatal("styles absent")
	}
}

func TestPickerDraftAndStaleState(t *testing.T) {
	c, err := config.Decode([]byte(ruleConfig))
	if err != nil {
		t.Fatal(err)
	}
	actions := make(chan Action, 8)
	s := screen{width: 80, height: 24, actions: actions, state: State{MicOptions: []string{"desk", "lav", "webcam", "off"}, Intent: c.VoiceIntent(), Revision: 4}}
	press := func(key tea.KeyPressMsg) {
		updated, _ := s.Update(key)
		s = updated.(screen)
	}
	for _, tab := range []int{0, 1, 2, 3} {
		s.tab = tab
		press(tea.KeyPressMsg{Code: '0'})
	}
	s.tab = 0
	press(tea.KeyPressMsg{Code: tea.KeyEnter})
	if s.picker == nil {
		t.Fatal("picker did not open")
	}
	for _, key := range []rune{'0', ' ', 'l', 'x', 'r'} {
		press(tea.KeyPressMsg{Code: key})
	}
	press(tea.KeyPressMsg{Code: tea.KeyDown})
	if len(actions) != 0 || s.state.Intent.Source != "desk" {
		t.Fatal("browsing dispatched an edit")
	}
	updated, _ := s.Update(stateMsg(s.state))
	s = updated.(screen)
	if s.picker == nil || s.picker.selected != 1 {
		t.Fatal("ordinary refresh changed draft")
	}
	press(tea.KeyPressMsg{Code: tea.KeyEscape})
	if s.picker != nil || len(actions) != 0 {
		t.Fatal("cancel dispatched")
	}
	press(tea.KeyPressMsg{Code: ' '})
	press(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(actions) != 0 {
		t.Fatal("same choice submitted")
	}
	press(tea.KeyPressMsg{Code: tea.KeyEnter})
	changed := s.state
	changed.Revision++
	updated, _ = s.Update(stateMsg(changed))
	s = updated.(screen)
	if s.picker != nil || !strings.Contains(s.pending, "reopen Source") || len(actions) != 0 {
		t.Fatal("stale draft not cancelled")
	}
	press(tea.KeyPressMsg{Code: tea.KeyEnter})
	press(tea.KeyPressMsg{Code: tea.KeyTab})
	if s.tab != 1 || s.picker != nil || len(actions) != 0 {
		t.Fatal("Tab did not cancel")
	}
}

func TestCompactLayoutAndPickerBounds(t *testing.T) {
	c, err := config.Decode([]byte(ruleConfig))
	if err != nil {
		t.Fatal(err)
	}
	c.Studio.Recording = &config.Recording{}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	s := screen{state: State{Intent: c.VoiceIntent(), MicOptions: []string{"desk", "lav", "webcam", "off"}}, actions: make(chan Action, 1)}
	want := []string{"record-computer", "record-mic", "record-tap", "record-vst", "record-loop", "record-start", "record-stop", "snippet-play", "snippet-stop"}
	for n, key := range want {
		if s.rules()[5+n].key != key {
			t.Fatal("recording order", s.rules())
		}
	}
	s.ruleAction("enter")
	for _, width := range []int{42, 80, 140} {
		for _, height := range []int{10, 24, 40} {
			s.width, s.height = width, height
			text := s.View().Content
			for _, row := range strings.Split(text, "\n") {
				if ansi.StringWidth(row) > width-1 {
					t.Fatal("picker overflow", width, height, row)
				}
			}
			for _, option := range []string{"Desk", "Lav", "Webcam", "Off"} {
				if !strings.Contains(text, option) {
					t.Fatal("picker option hidden", option)
				}
			}
		}
	}
	s.picker = nil
	s.width, s.height = 140, 40
	text := ansi.Strip(s.View().Content)
	for _, removed := range []string{"SOUND SNOOFER", "0 mic", "keeps running on exit", "before Element", "Choose a control", "Source: Desk"} {
		if strings.Contains(text, removed) {
			t.Fatal("obsolete chrome", removed)
		}
	}
	if !strings.Contains(text, "ACTIONS") || !strings.Contains(text, "Recording Mic Stage") {
		t.Fatal(text)
	}
}
