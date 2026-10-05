package control

import (
	"context"
	"strings"
	"testing"

	"sound-snoofer/internal/config"
)

func TestWorkerRejectsDisconnectedSelection(t *testing.T) {
	c, _ := config.Decode([]byte(ruleConfig))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deps := Dependencies{Open: func(string) (Client, error) { return &ruleClient{&fakeClient{}}, nil }, Save: func(string, config.Config, *config.Intent, string) (string, error) {
		t.Fatal("unavailable choice saved")
		return "", nil
	}}
	states, actions, done := make(chan State, 1), make(chan Action, 1), make(chan struct{})
	go Work(ctx, c, "unused", "", false, deps, actions, states, done)
	state := nextState(t, states)
	actions <- Action{Kind: editRule, ID: 1, Row: "output", Value: "speakers disconnected", Revision: state.Revision}
	state = nextState(t, states)
	if state.EditAck != 1 || !strings.Contains(state.EditError, "no longer connected") || state.Intent.PlaybackDevice != "" {
		t.Fatal(state.EditError)
	}
	cancel()
	<-done
}
