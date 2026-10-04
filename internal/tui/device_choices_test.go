package tui

import (
	"context"
	"strings"
	"testing"

	"sound-snoofer/internal/config"
)

func TestPickerDeviceRemovalAndOutputDraft(t *testing.T) {
	c, _ := config.Decode([]byte(ruleConfig))
	actions := make(chan Action, 2)
	s := screen{state: State{Intent: c.VoiceIntent(), MicOptions: []string{"desk", "lav", "off"}, OutputOptions: []string{"", "speakers"}}, actions: actions}
	s.openSource()
	s.pickerKey("down")
	s.state.MicOptions = []string{"off"}
	s.pickerKey("enter")
	if len(actions) != 0 || s.picker.options[0] != "off" {
		t.Fatal("disconnected source submitted")
	}
	s.picker = nil
	s.openChoice("output")
	s.pickerKey("down")
	s.pickerKey("enter")
	a := <-actions
	if a.Row != "output" || s.choices().PlaybackDevice != "speakers" {
		t.Fatal(a)
	}
	acknowledgeEdit(t, &s, a)
	s.openChoice("output")
	state := s.state
	state.OutputOptions = []string{""}
	m, _ := s.Update(stateMsg(state))
	s = m.(screen)
	if len(s.picker.options) != 1 || s.picker.selected != 0 || s.state.Intent.PlaybackDevice != "speakers" {
		t.Fatal("disconnection lost preference or retained option")
	}
}

func TestWorkerRejectsDisconnectedSelection(t *testing.T) {
	c, _ := config.Decode([]byte(ruleConfig))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deps := Dependencies{Open: func(string) (Client, error) { return &ruleClient{&fakeClient{}}, nil }, Save: func(string, config.Config, *config.Intent, string) (string, error) {
		t.Fatal("unavailable choice saved")
		return "", nil
	}}
	states, actions, done := make(chan State, 1), make(chan Action, 1), make(chan struct{})
	go work(ctx, c, "unused", "", false, deps, actions, states, done)
	state := nextState(t, states)
	actions <- Action{Kind: editRule, ID: 1, Row: "output", Value: "speakers disconnected", Revision: state.Revision}
	state = nextState(t, states)
	if state.EditAck != 1 || !strings.Contains(state.EditError, "no longer connected") || state.Intent.PlaybackDevice != "" {
		t.Fatal(state.EditError)
	}
	cancel()
	<-done
}
