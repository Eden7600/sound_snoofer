package control

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sound-snoofer/internal/config"
)

func TestIndependentMuteEditsFromSameSnapshot(t *testing.T) {
	for _, firstRow := range []string{"mic-mute", "speaker-mute"} {
		t.Run(firstRow, func(t *testing.T) {
			cfg, err := config.Decode([]byte(ruleConfig))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			actions, states, done := make(chan Action, 2), make(chan State, 1), make(chan struct{})
			deps := Dependencies{
				Open: func(string) (Client, error) { return nil, fmt.Errorf("offline fixture") },
				Save: func(_ string, _ config.Config, _ *config.Intent, _ string) (string, error) { return "saved", nil },
			}
			go Work(ctx, cfg, filepath.Join(t.TempDir(), "audio"), "", false, deps, actions, states, done)
			t.Cleanup(func() {
				cancel()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Error("audio worker did not stop")
				}
			})
			initial := nextState(t, states)
			otherRow := "speaker-mute"
			if firstRow == otherRow {
				otherRow = "mic-mute"
			}
			for _, row := range []string{firstRow, otherRow} {
				generation := initial.MicMuteGeneration
				if row == "speaker-mute" {
					generation = initial.PlaybackMuteGeneration
				}
				actions <- Action{Kind: Edit, Row: row, Value: "true", Revision: initial.Revision, MuteGeneration: generation}
			}
			for {
				state := nextState(t, states)
				if state.Intent != nil && state.Intent.MicMuted && state.Intent.PlaybackMuted {
					return
				}
				if strings.Contains(state.Notice, "Stale") {
					t.Fatalf("independent second preference was rejected: %s", state.Notice)
				}
			}
		})
	}
}
