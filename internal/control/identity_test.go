package control

import (
	"context"
	"slices"
	"strings"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/windowsaudio"
)

// fakeDefaults reports a fixed endpoint inventory and acknowledges requests.
func fakeDefaults(endpoints []windowsaudio.Endpoint) func(context.Context) (chan windowsaudio.Request, chan windowsaudio.Result, <-chan struct{}) {
	return func(ctx context.Context) (chan windowsaudio.Request, chan windowsaudio.Result, <-chan struct{}) {
		requests, results, done := make(chan windowsaudio.Request, 1), make(chan windowsaudio.Result, 1), make(chan struct{})
		go func() {
			defer close(done)
			for {
				select {
				case <-ctx.Done():
					return
				case r := <-requests:
					if r.Ack != nil {
						close(r.Ack)
					}
					select {
					case results <- windowsaudio.Result{Endpoints: endpoints}:
					default:
					}
				}
			}
		}()
		return requests, results, done
	}
}

// A playback entry chosen by identity matches its endpoint's current name.
func TestWorkerResolvesDeviceIdentity(t *testing.T) {
	raw := strings.Replace(ruleConfig, `{"driver":"wdm","pattern":"speakers"}`, `{"driver":"wdm","id":"{endpoint}","name":"Speakers (old name)"}`, 1)
	c, err := config.Decode([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &ruleClient{&fakeClient{}}
	deps := Dependencies{Open: func(string) (Client, error) { return client, nil }, Load: func(string) (config.Config, error) { return c, nil },
		StartDefaults: fakeDefaults([]windowsaudio.Endpoint{{ID: "{endpoint}", Name: "speakers"}})}
	states := make(chan State, 1)
	done := make(chan struct{})
	actions := make(chan Action, 1)
	go Work(ctx, c, "unused", "", false, deps, actions, states, done)
	for n := 0; n < 20; n++ {
		if s := nextState(t, states); slices.Contains(s.OutputOptions, "speakers") {
			cancel()
			<-done
			return
		}
		// The fixture polls rarely; a reload re-plans and publishes.
		actions <- Reload
	}
	t.Fatal("identity entry did not resolve to the endpoint's current name")
}
