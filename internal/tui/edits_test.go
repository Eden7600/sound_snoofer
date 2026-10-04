package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"sound-snoofer/internal/config"
)

func acknowledgeEdit(t *testing.T, s *screen, action Action) {
	t.Helper()
	s.state.Intent = s.state.Intent.Clone()
	if err := editBatch(s.state.Intent, action); err != nil {
		t.Fatal(err)
	}
	s.state.Revision++
	s.state.EditAck = action.ID
	s.acceptEditState()
}

func TestEditsStayResponsiveUntilAcknowledged(t *testing.T) {
	c, err := config.Decode([]byte(ruleConfig))
	if err != nil {
		t.Fatal(err)
	}
	actions := make(chan Action, 8)
	s := screen{state: State{Intent: c.VoiceIntent(), Revision: 1}, actions: actions, width: 100, height: 30}
	s.queue(Action{Kind: editRule, Row: "monitor", Value: "pre"})
	first := <-actions
	s.selected = 2
	s.ruleAction("enter") // Pre -> Post, from the draft rather than Off.
	s.queue(Action{Kind: editRule, Row: "source", Value: "lav"})
	if s.choices().Monitor != "post" || s.choices().Source != "lav" || len(actions) != 0 {
		t.Fatal("edits stalled or submitted concurrently")
	}
	if s.state.Intent.Monitor != "off" {
		t.Fatal("draft mutated authoritative intent")
	}
	refresh := s.state
	refresh.Revision++
	m, _ := s.Update(stateMsg(refresh))
	s = m.(screen)
	if s.choices().Monitor != "post" || s.pending != "Queued" {
		t.Fatal("refresh lost draft")
	}
	m, _ = s.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	s = m.(screen)
	if s.tab != 1 {
		t.Fatal("navigation blocked")
	}
	s.queue(ToggleLive)
	if len(actions) != 0 {
		t.Fatal("live toggle reordered edits")
	}
	acknowledgeEdit(t, &s, first)
	batch := <-actions
	if batch.Revision != s.state.Revision || len(batch.Edits) != 2 {
		t.Fatal(batch)
	}
	acknowledgeEdit(t, &s, batch)
	if s.inflight != 0 || s.draft != nil || s.state.Intent.Monitor != "post" || s.state.Intent.Source != "lav" {
		t.Fatal("latest choices lost")
	}
}

func TestEditRejectionAndQueueBound(t *testing.T) {
	c, _ := config.Decode([]byte(ruleConfig))
	actions := make(chan Action, 1)
	s := screen{state: State{Intent: c.VoiceIntent()}, actions: actions}
	s.queue(Action{Kind: editRule, Row: "source", Value: "lav"})
	first := <-actions
	for n := 0; n < maxDeferredEdits; n++ {
		s.queue(Action{Kind: editRule, Row: "monitor", Value: "pre"})
	}
	s.queue(Action{Kind: editRule, Row: "source", Value: "off"})
	if s.choices().Source != "lav" || !strings.Contains(s.pending, "full") {
		t.Fatal("queue limit lost accepted edits")
	}
	s.state.EditAck, s.state.EditError = first.ID, "disk full"
	s.acceptEditState()
	if s.draft != nil || s.inflight != 0 || len(s.deferredEdits) != 0 || !strings.Contains(s.pending, "disk full") {
		t.Fatal("failure not reverted visibly")
	}
}

func TestWorkerBatchSavesOnceAndAcknowledgesFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			c, _ := config.Decode([]byte(ruleConfig))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := &ruleClient{&fakeClient{}}
			saves := 0
			deps := Dependencies{Open: func(string) (Client, error) { return client, nil }, Save: func(_ string, _ config.Config, i *config.Intent, _ string) (string, error) {
				saves++
				if i.Source != "lav" || i.Monitor != "post" {
					t.Fatal("batch incomplete", i)
				}
				if fail {
					return "", fmt.Errorf("disk full")
				}
				return "saved", nil
			}}
			states, actions, done := make(chan State, 1), make(chan Action, 1), make(chan struct{})
			go work(ctx, c, "unused", "", false, deps, actions, states, done)
			state := nextState(t, states)
			actions <- Action{Kind: editRule, ID: 1, Revision: state.Revision, Edits: []settingEdit{{"source", "lav"}, {"monitor", "pre"}, {"monitor", "post"}}}
			state = nextState(t, states)
			if state.EditAck != 1 || (state.EditError != "") != fail || saves != 1 {
				t.Fatal(state.EditAck, state.EditError, saves)
			}
			if fail && state.Intent.Source != "desk" {
				t.Fatal("failed save changed intent")
			}
			cancel()
			<-done
			if client.writes.Load() != 0 {
				t.Fatal("preview wrote")
			}
		})
	}
}
