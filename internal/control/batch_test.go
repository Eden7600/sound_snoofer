package control

import (
	"context"
	"fmt"

	"testing"

	"sound-snoofer/internal/config"
)

// Pre -> Post, from the draft rather than Off.

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
			go Work(ctx, c, "unused", "", false, deps, actions, states, done)
			state := nextState(t, states)
			actions <- Action{Kind: editRule, ID: 1, Revision: state.Revision, Edits: []SettingEdit{{Row: "source", Value: "lav"}, {Row: "monitor", Value: "pre"}, {Row: "monitor", Value: "post"}}}
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
