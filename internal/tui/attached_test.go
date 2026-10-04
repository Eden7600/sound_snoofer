package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestAttachedQuitLeavesActorRunning(t *testing.T) {
	actorCtx, stopActor := context.WithCancel(context.Background())
	defer stopActor()
	client := &fakeClient{}
	actions, states, done := StartWorker(actorCtx, cfg(t), "config", "", false, Dependencies{Open: func(string) (Client, error) { return client, nil }})
	initial := nextState(t, states)
	viewCtx, closeView := context.WithCancel(actorCtx)
	view := screen{ctx: viewCtx, cancel: closeView, state: initial, attached: true, width: 100, height: 30, actions: actions, states: states}
	if text := view.View().Content; !strings.Contains(text, "Q close") || strings.Contains(text, "Q quit") {
		t.Fatal("attached view mislabels exit")
	}
	view.Update(tea.KeyPressMsg{Code: 'q'})
	if viewCtx.Err() == nil || actorCtx.Err() != nil {
		t.Fatal("wrong lifetime canceled")
	}
	actions <- Refresh
	if !nextState(t, states).Connected || client.closed.Load() != 0 {
		t.Fatal("detached view stopped worker")
	}
	stopActor()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("actor did not stop")
	}
	if client.closed.Load() != 1 {
		t.Fatal("backend not closed once")
	}
}
