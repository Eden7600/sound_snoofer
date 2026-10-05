package control

import (
	"context"
	"fmt"
	"path/filepath"
	"sound-snoofer/internal/config"
	"testing"
	"time"
)

func TestHardwareAckCannotOverwriteControls(t *testing.T) {
	c := config.Config{Studio: &config.Studio{Voice: &config.Voice{Source: "desk", Mode: "direct", Monitor: "off"}}, Intent: &config.Intent{Version: 1, Enabled: true, Source: "desk", Mode: "direct", Monitor: "off", Playback: map[string]bool{}}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	actions := make(chan Action, 8)
	states := make(chan State, 1)
	done := make(chan struct{})
	saves := 0
	deps := Dependencies{Open: func(string) (Client, error) { return nil, fmt.Errorf("offline fixture") }, Save: func(string, config.Config, *config.Intent, string) (string, error) { saves++; return "saved", nil }}
	go Work(ctx, c, filepath.Join(t.TempDir(), "config"), "", false, deps, actions, states, done)
	next := func() State {
		select {
		case s := <-states:
			return s
		case <-time.After(2 * time.Second):
			t.Fatal("worker did not publish")
			return State{}
		}
	}
	s := next()
	actions <- Action{Kind: Edit, ID: 10, Revision: s.Revision, Row: "mic-mute", Value: "true"}
	s = next()
	if s.EditAck != 10 || s.EditError != "" {
		t.Fatal(s)
	}
	actions <- Action{Kind: Edit, Origin: "deck", ID: 1, Revision: s.Revision, Row: "mic-mute", Value: "not-bool"}
	s = next()
	if s.EditAck != 10 || s.EditError != "" || s.Acks["deck"].Error == "" {
		t.Fatal("ack cross-talk", s.Acks, s.EditAck, s.EditError)
	}
	actions <- Action{Kind: Edit, Origin: "deck", ID: 2, Revision: s.Revision, Row: "mic-mute", Value: "false"}
	s = next()
	if s.Acks["deck"].Error != "" || s.Intent.MicMuted {
		t.Fatal(s.Acks)
	}
	actions <- Action{Kind: Edit, Origin: "deck", ID: 2, Revision: s.Revision, Row: "mic-mute", Value: "true"}
	s = next()
	if s.Intent.MicMuted || saves != 2 {
		t.Fatal("duplicate action replayed")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker leaked")
	}
}
