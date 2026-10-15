package control

import (
	"context"
	"strings"
	"testing"

	"sound-snoofer/internal/config"
)

func TestWorkerRejectsDisconnectedSelection(t *testing.T) {
	for _, row := range []string{"output", "normal-output", "vr-profile-output"} {
		t.Run(row, func(t *testing.T) {
			c, _ := config.Decode([]byte(ruleConfig))
			c.VR = &config.VR{Input: 4, Headsets: []config.Headset{{ID: "headset", Playback: "^VR Phones$"}}}
			if err := c.VR.Validate(); err != nil {
				t.Fatal(err)
			}
			c.Intent = c.VoiceIntent()
			c.Intent.VRProfile = &config.ProfileChoices{Source: "auto", Mode: "direct", Monitor: "off"}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			deps := Dependencies{Open: func(string) (Client, error) { return &ruleClient{&fakeClient{}}, nil }, Save: func(string, config.Config, *config.Intent, string) (string, error) {
				t.Fatal("unavailable choice saved")
				return "", nil
			}}
			states, actions, done := make(chan State, 1), make(chan Action, 1), make(chan struct{})
			go Work(ctx, c, "unused", "", false, deps, actions, states, done)
			state := nextState(t, states)
			actions <- Action{Kind: editRule, ID: 1, Row: row, Value: "speakers disconnected", Revision: state.Revision}
			state = nextState(t, states)
			if state.EditAck != 1 || !strings.Contains(state.EditError, "no longer connected") {
				t.Fatal(state.EditError)
			}
			cancel()
			<-done
		})
	}
}
