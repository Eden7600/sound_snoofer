package tui

import (
	"context"
	"sound-snoofer/internal/config"
	"strings"
	"testing"
)

func TestReloadRejectsUnsupportedEditionTargets(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	original := cfg(t)
	replacement := cfg(t)
	replacement.Routes[0].Target = "A4"
	client := &fakeClient{}
	deps := Dependencies{Open: func(string) (Client, error) { return client, nil }, Load: func(string) (config.Config, error) { return replacement, nil }}
	actions := make(chan Action, 1)
	states := make(chan State, 1)
	done := make(chan struct{})
	go work(ctx, original, "config", "", false, deps, actions, states, done)
	nextState(t, states)
	actions <- Reload
	s := nextState(t, states)
	if !strings.Contains(s.Notice, "unavailable") || !s.Connected || s.Plan == nil || s.Plan.Decisions[0].Target != "A1" {
		t.Fatal("replaced valid config with unsupported target", s)
	}
	cancel()
	<-done
}
