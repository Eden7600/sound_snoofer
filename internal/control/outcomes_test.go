package control

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/voicemeeter"
	"sound-snoofer/snoofer"
)

func outcomeWorker(t *testing.T, cfg config.Config, path string, deps Dependencies) (chan Action, chan State, context.CancelFunc) {
	t.Helper()
	if deps.Open == nil {
		deps.Open = func(string) (Client, error) { return nil, errors.New("offline fixture") }
	}
	if deps.Save == nil {
		deps.Save = func(string, config.Config, *config.Intent, string) (string, error) { return "saved", nil }
	}
	if deps.Load == nil {
		deps.Load = func(string) (config.Config, error) { return cfg, nil }
	}
	ctx, cancel := context.WithCancel(context.Background())
	actions, states, done := make(chan Action, 32), make(chan State, 1), make(chan struct{})
	go Work(ctx, cfg, path, "", false, deps, actions, states, done)
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("worker did not stop")
		}
	})
	return actions, states, cancel
}

func testRuleConfig(t *testing.T) config.Config {
	t.Helper()
	cfg, err := config.Decode([]byte(ruleConfig))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func actionOutcome(t *testing.T, actions chan<- Action, a Action) snoofer.Outcome {
	t.Helper()
	results := make(chan snoofer.Outcome, 1)
	a.Complete = func(o snoofer.Outcome) { results <- o }
	actions <- a
	select {
	case result := <-results:
		return result
	case <-time.After(time.Second):
		t.Fatal("action did not complete")
		return snoofer.Outcome{}
	}
}

func TestMuteGenerationABA(t *testing.T) {
	actions, states, _ := outcomeWorker(t, testRuleConfig(t), filepath.Join(t.TempDir(), "audio"), Dependencies{})
	initial := nextState(t, states)
	state := initial
	for _, value := range []string{"true", "false"} {
		a := Action{Kind: Edit, Row: "mic-mute", Value: value, Revision: state.Revision}
		if result := actionOutcome(t, actions, a); result.Status != snoofer.Processed {
			t.Fatal(result)
		}
		state = nextState(t, states)
	}
	if state.Intent.MicMuted || state.MicMuteGeneration != initial.MicMuteGeneration+2 || state.PlaybackMuteGeneration != initial.PlaybackMuteGeneration {
		t.Fatal(state)
	}
	old := Action{Kind: Edit, Row: "mic-mute", Value: "true", Revision: initial.Revision, MuteGeneration: initial.MicMuteGeneration}
	if result := actionOutcome(t, actions, old); result.Status != snoofer.Rejected {
		t.Fatal("ABA accepted", result)
	}
}

func TestMuteGenerationReloadAndResetBoundaries(t *testing.T) {
	for _, kind := range []ActionKind{reload, resetChoices} {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			actions, states, _ := outcomeWorker(t, testRuleConfig(t), filepath.Join(t.TempDir(), "audio"), Dependencies{})
			initial := nextState(t, states)
			if result := actionOutcome(t, actions, Action{Kind: kind, Revision: initial.Revision}); result.Status != snoofer.Processed {
				t.Fatal(result)
			}
			state := nextState(t, states)
			if state.MicMuteGeneration <= initial.MicMuteGeneration || state.PlaybackMuteGeneration <= initial.PlaybackMuteGeneration {
				t.Fatal("boundary kept old generations", state)
			}
			old := Action{Kind: Edit, Row: "speaker-mute", Value: "true", Revision: initial.Revision, MuteGeneration: initial.PlaybackMuteGeneration}
			if result := actionOutcome(t, actions, old); result.Status != snoofer.Rejected {
				t.Fatal(result)
			}
		})
	}
}

func TestMuteGenerationIntentLifetimeAndScope(t *testing.T) {
	intent := &config.Intent{}
	g := newMuteGenerations(intent)
	old := Action{Kind: Edit, Row: "mic-mute", Value: "true", MuteGeneration: g.mic}
	g.update(nil, false)
	g.update(intent, false)
	if g.matches(old) || g.mic != 3 || g.playback != 3 {
		t.Fatal("recreated intent accepted old action", g)
	}
	for _, a := range []Action{
		{Kind: Edit, Row: "mode", MuteGeneration: g.mic},
		{Kind: resetChoices, Row: "mic-mute", MuteGeneration: g.mic},
	} {
		if g.matches(a) {
			t.Fatal("independent policy escaped explicit controls", a)
		}
	}
}

func TestIndependentMutePreservesExternalStateCAS(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audio")
	cfg := config.LoadChoices(path, testRuleConfig(t))
	actions, states, _ := outcomeWorker(t, cfg, path, Dependencies{Save: config.SaveIntent})
	initial := nextState(t, states)
	external := cfg.VoiceIntent()
	external.PlaybackMuted = true
	if _, err := config.SaveIntent(path, cfg, external, cfg.StateToken); err != nil {
		t.Fatal(err)
	}
	result := actionOutcome(t, actions, Action{Kind: Edit, Row: "mic-mute", Value: "true", Revision: initial.Revision, MuteGeneration: initial.MicMuteGeneration})
	if result.Status != snoofer.Failed || !strings.Contains(result.Detail, "changed") {
		t.Fatal(result)
	}
	saved := config.LoadChoices(path, cfg).VoiceIntent()
	if saved.MicMuted || !saved.PlaybackMuted {
		t.Fatal("external saved choices overwritten", saved)
	}
}

func TestAudioOutcomeWaitsForSaveAndSettlesQueuedStop(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	deps := Dependencies{Save: func(string, config.Config, *config.Intent, string) (string, error) {
		close(started)
		<-release
		return "", errors.New("disk full")
	}}
	actions, states, cancel := outcomeWorker(t, testRuleConfig(t), filepath.Join(t.TempDir(), "audio"), deps)
	state := nextState(t, states)
	first, second := make(chan snoofer.Outcome, 1), make(chan snoofer.Outcome, 1)
	actions <- Action{Kind: Edit, Row: "mic-mute", Value: "true", Revision: state.Revision, Complete: func(o snoofer.Outcome) { first <- o }}
	<-started
	select {
	case result := <-first:
		t.Fatal("save returned enqueue acknowledgment", result)
	default:
	}
	actions <- Action{Kind: Edit, Row: "speaker-mute", Value: "true", Revision: state.Revision, Complete: func(o snoofer.Outcome) { second <- o }}
	cancel()
	close(release)
	if result := <-first; result.Status != snoofer.Failed || result.Detail != "disk full" {
		t.Fatal(result)
	}
	select {
	case result := <-second:
		if result.Status != snoofer.Unknown {
			t.Fatal(result)
		}
	case <-time.After(time.Second):
		t.Fatal("queued action leaked on stop")
	}
}

func TestAudioOutcomesStaleAndReloadFailure(t *testing.T) {
	actions, states, _ := outcomeWorker(t, testRuleConfig(t), filepath.Join(t.TempDir(), "audio"), Dependencies{Load: func(string) (config.Config, error) { return config.Config{}, errors.New("read denied") }})
	initial := nextState(t, states)
	for _, kind := range []ActionKind{gain, startRecording, restartEngine, editRule} {
		if result := actionOutcome(t, actions, Action{Kind: kind, Row: "mic-mute", Value: "true", Revision: initial.Revision - 1}); result.Status != snoofer.Rejected {
			t.Fatal(kind, result)
		}
		nextState(t, states)
	}
	if result := actionOutcome(t, actions, Reload); result.Status != snoofer.Failed || !strings.Contains(result.Detail, "read denied") {
		t.Fatal(result)
	}
	state := nextState(t, states)
	if state.MicMuteGeneration == initial.MicMuteGeneration {
		t.Fatal("failed reload retained captured preference lifetime")
	}
}

type outcomeRestartClient struct {
	*recorderClient
	err   error
	calls atomic.Int32
}

func (c *outcomeRestartClient) RestartEngine() error                            { c.calls.Add(1); return c.err }
func (c *outcomeRestartClient) SetCallback(bool, *voicemeeter.InsertHook) error { return nil }

func TestAudioNativeRestartOutcomeDoesNotReplay(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status snoofer.OutcomeStatus
	}{
		{"submitted", nil, snoofer.Submitted},
		{"failure", errors.New("restart refused"), snoofer.Failed},
		{"timeout", context.DeadlineExceeded, snoofer.Unknown},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &outcomeRestartClient{recorderClient: &recorderClient{ruleClient: &ruleClient{&fakeClient{}}, transport: "stop"}, err: test.err}
			deps := Dependencies{Open: func(string) (Client, error) { return client, nil }, Acquire: func() (func(), error) { return func() {}, nil }}
			actions, states, _ := outcomeWorker(t, testRuleConfig(t), filepath.Join(t.TempDir(), "audio"), deps)
			nextState(t, states)
			if result := actionOutcome(t, actions, ToggleLive); result.Status != snoofer.Processed {
				t.Fatal(result)
			}
			state := nextState(t, states)
			result := actionOutcome(t, actions, Action{Kind: Restart, Revision: state.Revision})
			if result.Status != test.status {
				t.Fatal(result)
			}
			if result := actionOutcome(t, actions, Refresh); result.Status != snoofer.Processed {
				t.Fatal(result)
			}
			if client.calls.Load() != 1 {
				t.Fatal("native restart replayed", client.calls.Load())
			}
		})
	}
}
