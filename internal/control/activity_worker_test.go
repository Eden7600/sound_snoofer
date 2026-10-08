package control

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"sound-snoofer/internal/config"
)

// activityClient reports a silent lav (strip 1) and a speaking desk mic.
type activityClient struct{ *ruleClient }

func (c *activityClient) InputLevels(strips []int) map[int]float32 {
	levels := map[int]float32{}
	for _, strip := range strips {
		levels[strip] = map[int]float32{0: 0.1, 1: 0.00001}[strip]
	}
	return levels
}

func TestWorkerSkipsSilentMicrophone(t *testing.T) {
	raw := strings.Replace(ruleConfig, `"version":1,`, `"version":1,"profiles":{"microphones":["lav","desk"],"activity":{"check":2,"silent_after_s":2}},`, 1)
	c, err := config.Decode([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	c.Intent = c.VoiceIntent()
	c.Intent.Source = "auto"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &activityClient{&ruleClient{&fakeClient{}}}
	deps := Dependencies{Open: func(string) (Client, error) { return client, nil }, Load: func(string) (config.Config, error) { return c, nil }}
	states := make(chan State, 1)
	done := make(chan struct{})
	go Work(ctx, c, "unused", "", false, deps, make(chan Action), states, done)
	effective := func(s State) string {
		if s.Plan == nil || s.Plan.Topology == nil || s.Plan.Topology.Voice == nil {
			return ""
		}
		return s.Plan.Topology.Voice.Effective
	}
	deadline := time.Now().Add(6 * time.Second)
	sawLav := false
	for time.Now().Before(deadline) {
		s := nextState(t, states)
		sawLav = sawLav || effective(s) == "lav"
		if effective(s) == "desk" {
			if !sawLav || !slices.Equal(s.SilentMics, []string{"lav"}) {
				t.Fatal("desk chosen before the lav fell silent", s.SilentMics)
			}
			cancel()
			<-done
			return
		}
	}
	t.Fatal("silent lav still selected")
}
