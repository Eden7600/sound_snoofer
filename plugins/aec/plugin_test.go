package aec

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	engine "sound-snoofer/internal/aec"
	"sound-snoofer/internal/voicemeeter"
	"sound-snoofer/plugins/audio"
	"sound-snoofer/snoofer"
)

type fakeEngine struct {
	mu      sync.Mutex
	configs []engine.Config
	stats   engine.Stats
	closed  bool
	resets  int
}

func (e *fakeEngine) Configure(c engine.Config) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.configs = append(e.configs, c)
	return nil
}
func (e *fakeEngine) Stats() (engine.Stats, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stats, nil
}
func (e *fakeEngine) Reset() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.resets++
	return nil
}
func (e *fakeEngine) Hook() voicemeeter.InsertHook {
	return voicemeeter.InsertHook{Input: 1, Output: 2, Context: 3}
}
func (e *fakeEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closed = true
	return nil
}
func (e *fakeEngine) last() engine.Config {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.configs[len(e.configs)-1]
}

type fakeMixer struct {
	mu         sync.Mutex
	targets    audio.EchoTargets
	hook       *voicemeeter.InsertHook
	refuseNil  bool // Removal stays unconfirmed.
	insertions int
}

func (m *fakeMixer) EchoTargets() audio.EchoTargets {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.targets
}
func (m *fakeMixer) SetEchoInsert(ctx context.Context, hook *voicemeeter.InsertHook) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if hook == nil && m.refuseNil {
		return errors.New("echo insert release unconfirmed")
	}
	if hook != nil {
		m.insertions++
	}
	m.hook = hook
	return nil
}

var speakers = audio.EchoTargets{Mic: [2]int{0, 1}, Reference: [8]int{8, 9, 10, 11, 12, 13, 14, 15}, Playback: "Speakers (Realtek)"}

func newWorker(settings Settings, m *fakeMixer, e *fakeEngine) *worker {
	raw, _ := json.Marshal(settings)
	services := snoofer.Services{Controls: snoofer.NewControls(), SaveSettings: func(string, json.RawMessage, json.RawMessage) error { return nil }}
	return &worker{services: services, mixer: m, open: func(string) (canceller, error) { return e, nil }, raw: raw, settings: settings, requests: make(chan snoofer.Request, 8)}
}

func control(t *testing.T, w *worker, id string) snoofer.Control {
	t.Helper()
	w.publish()
	for _, c := range w.services.Controls.Snapshot() {
		if c.ID == id {
			return c
		}
	}
	t.Fatal("missing", id)
	return snoofer.Control{}
}

func TestDecide(t *testing.T) {
	custom := []string{"(?i)monitor"}
	for name, test := range map[string]struct {
		settings Settings
		targets  audio.EchoTargets
		active   bool
		reason   string
	}{
		"auto speakers":           {Settings{}, speakers, true, ""},
		"auto headphones":         {Settings{}, audio.EchoTargets{Playback: "Headphones (USB)"}, false, "Headphones"},
		"auto headset":            {Settings{}, audio.EchoTargets{Playback: "Arctis Headset Game"}, false, "Headphones"},
		"on headphones":           {Settings{Mode: "on"}, audio.EchoTargets{Playback: "Headphones (USB)"}, true, ""},
		"off":                     {Settings{Mode: "off"}, speakers, false, "Off"},
		"mic off":                 {Settings{Mode: "on"}, audio.EchoTargets{Reason: "Mic off"}, false, "Mic off"},
		"custom speaker match":    {Settings{SpeakerOutputs: custom}, audio.EchoTargets{Playback: "Studio Monitor 5"}, true, ""},
		"custom speaker mismatch": {Settings{SpeakerOutputs: custom}, speakers, false, "Headphones"},
	} {
		patterns, err := test.settings.speakers()
		if err != nil {
			t.Fatal(err)
		}
		active, reason := decide(test.settings, patterns, test.targets)
		if active != test.active || reason != test.reason {
			t.Errorf("%s: %v %q", name, active, reason)
		}
	}
}

func TestActivationConfiguresThenHooks(t *testing.T) {
	m := &fakeMixer{targets: speakers}
	e := &fakeEngine{stats: engine.Stats{Active: true, SampleRate: 48000, ERLEKnown: true, ERLE: 32.4, DelayKnown: true, DelayMs: 54}}
	w := newWorker(Settings{Strength: "gentle"}, m, e)
	w.step(context.Background())
	if m.hook == nil || *m.hook != e.Hook() || !w.hooked {
		t.Fatal("hook not installed", m.hook)
	}
	if got := e.last(); got.Mic != speakers.Mic || got.Reference != speakers.Reference || got.Strength != engine.Gentle || got.Bypass {
		t.Fatal(got)
	}
	status := control(t, w, "aec.status")
	if status.Value != "Active" || status.Status != "−32 dB echo · 54 ms" {
		t.Fatal(status.Value, status.Status)
	}
	// Steady state: no reconfiguration and no second installation.
	w.step(context.Background())
	if len(e.configs) != 1 || m.insertions != 1 {
		t.Fatal(len(e.configs), m.insertions)
	}
	// A new mic strip reconfigures in place.
	m.targets.Mic = [2]int{2, 3}
	w.step(context.Background())
	if e.last().Mic != [2]int{2, 3} || m.insertions != 1 {
		t.Fatal(e.last(), m.insertions)
	}
}

func TestHeadphonesBypassAndRemoveHook(t *testing.T) {
	m := &fakeMixer{targets: speakers}
	e := &fakeEngine{stats: engine.Stats{Active: true, SampleRate: 48000}}
	w := newWorker(Settings{}, m, e)
	w.step(context.Background())
	m.targets.Playback = "Headphones (USB)"
	w.step(context.Background())
	if !e.last().Bypass || m.hook != nil || w.hooked {
		t.Fatal(e.last(), m.hook, w.hooked)
	}
	if status := control(t, w, "aec.status"); status.Value != "Idle" || status.Status != "Headphones" {
		t.Fatal(status.Value, status.Status)
	}
}

func TestStatusStates(t *testing.T) {
	m := &fakeMixer{targets: speakers}
	e := &fakeEngine{stats: engine.Stats{SampleRate: 44100}}
	w := newWorker(Settings{}, m, e)
	w.step(context.Background())
	if status := control(t, w, "aec.status"); status.Value != "Idle" || status.Status != "Needs 48 kHz" {
		t.Fatal(status.Value, status.Status)
	}
	e.stats = engine.Stats{Failed: true}
	w.step(context.Background())
	if status := control(t, w, "aec.status"); status.Value != "Error" {
		t.Fatal(status.Value)
	}
	e.stats = engine.Stats{}
	m.targets = audio.EchoTargets{Reason: "Mic off"}
	w.step(context.Background())
	if status := control(t, w, "aec.status"); status.Value != "Idle" || status.Status != "Mic off" {
		t.Fatal(status.Value, status.Status)
	}
	w.settings.Mode = "off"
	w.step(context.Background())
	if status := control(t, w, "aec.status"); status.Value != "Off" {
		t.Fatal(status.Value)
	}
}

func TestEngineLoadsOnlyWhenNeeded(t *testing.T) {
	m := &fakeMixer{targets: audio.EchoTargets{Reason: "Mic off"}}
	opened := 0
	w := newWorker(Settings{}, m, &fakeEngine{})
	w.open = func(string) (canceller, error) {
		opened++
		return nil, errors.New("missing")
	}
	w.step(context.Background())
	if opened != 0 {
		t.Fatal("opened while idle")
	}
	m.targets = speakers
	w.step(context.Background())
	if status := control(t, w, "aec.status"); opened != 1 || status.Value != "Error" || m.hook != nil {
		t.Fatal(opened, status.Value, m.hook)
	}
}

func TestModeAndStrengthSave(t *testing.T) {
	w := newWorker(Settings{}, &fakeMixer{targets: speakers}, &fakeEngine{})
	var saved Settings
	w.services.SaveSettings = func(id string, _, next json.RawMessage) error {
		if id != "aec" {
			t.Fatal(id)
		}
		return json.Unmarshal(next, &saved)
	}
	w.handle(snoofer.Request{ID: "aec.mode", Value: "off"})
	w.handle(snoofer.Request{ID: "aec.strength", Value: "balanced"})
	if saved.Mode != "off" || saved.Strength != "balanced" || w.settings.Mode != "off" || w.settings.Strength != "balanced" {
		t.Fatal(saved, w.settings)
	}
	w.handle(snoofer.Request{ID: "aec.mode", Value: "loud"})
	if mode := control(t, w, "aec.mode"); mode.Value != "off" || mode.Status == "" {
		t.Fatal(mode.Value, mode.Status)
	}
}

func TestStopFreesEngineOnlyAfterRemoval(t *testing.T) {
	for _, confirmed := range []bool{true, false} {
		m := &fakeMixer{targets: speakers, refuseNil: !confirmed}
		e := &fakeEngine{stats: engine.Stats{Active: true, SampleRate: 48000}}
		services := snoofer.Services{Controls: snoofer.NewControls()}
		i, err := start(context.Background(), services, json.RawMessage(`{}`), m, func(string) (canceller, error) { return e, nil }, 10*time.Millisecond)
		if err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(time.Second)
		for {
			m.mu.Lock()
			hooked := m.hook != nil
			m.mu.Unlock()
			if hooked {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("hook never installed")
			}
			time.Sleep(5 * time.Millisecond)
		}
		err = i.Stop(context.Background())
		e.mu.Lock()
		closed := e.closed
		e.mu.Unlock()
		if confirmed && (err != nil || !closed || m.hook != nil) {
			t.Fatal("confirmed removal", err, closed)
		}
		if !confirmed && (err == nil || closed) {
			t.Fatal("unconfirmed removal freed the engine", err, closed)
		}
	}
}

func TestFailureResetsOncePerNewStream(t *testing.T) {
	m := &fakeMixer{targets: speakers}
	m.targets.Stream = 1
	e := &fakeEngine{stats: engine.Stats{Active: true, SampleRate: 48000}}
	w := newWorker(Settings{}, m, e)
	w.step(context.Background())
	// A failure on the armed stream stays latched, even when first seen before
	// the new stream's count is published.
	e.stats = engine.Stats{Failed: true, Reason: "missing output callback"}
	w.step(context.Background())
	w.step(context.Background())
	if e.resets != 0 {
		t.Fatal("reset on the armed stream", e.resets)
	}
	if status := control(t, w, "aec.status"); status.Value != "Error" || status.Status != "Engine error (missing output callback); mic passes through" {
		t.Fatal(status.Value, status.Status)
	}
	m.targets.Stream = 2
	w.step(context.Background())
	if e.resets != 1 {
		t.Fatal("no reset for the new stream", e.resets)
	}
	// The engine keeps failing on the new stream: it stays latched there.
	w.step(context.Background())
	w.step(context.Background())
	if e.resets != 1 {
		t.Fatal("reset flapped on one stream", e.resets)
	}
	e.stats = engine.Stats{Active: true, SampleRate: 48000}
	w.step(context.Background())
	if status := control(t, w, "aec.status"); status.Value != "Active" {
		t.Fatal(status.Value, status.Status)
	}
}

func TestFailureWithoutHookIsNotReset(t *testing.T) {
	m := &fakeMixer{targets: audio.EchoTargets{Reason: "Mic off", Stream: 3}}
	e := &fakeEngine{stats: engine.Stats{Failed: true}}
	w := newWorker(Settings{}, m, e)
	w.engine = e
	w.step(context.Background())
	if e.resets != 0 {
		t.Fatal("reset without an installed hook", e.resets)
	}
}

func TestEngineSwitchOwnershipAndPersistence(t *testing.T) {
	ctx := context.Background()
	m := &fakeMixer{targets: speakers}
	old := &fakeEngine{stats: engine.Stats{Active: true, SampleRate: 48000}}
	neural := &fakeEngine{}
	w := newWorker(Settings{}, m, old)
	w.step(ctx)
	w.open = func(choice string) (canceller, error) {
		if choice != "localvqe-aec" || !old.closed || m.hook != nil {
			t.Fatal("unsafe switch", choice, old.closed, m.hook)
		}
		return neural, nil
	}
	w.services.SaveSettings = func(string, json.RawMessage, json.RawMessage) error { return errors.New("disk full") }
	w.handle(snoofer.Request{ID: "aec.engine", Value: "localvqe-aec"})
	w.step(ctx)
	if w.settings.engine() != "aec3" || old.closed {
		t.Fatal("applied an unsaved engine")
	}
	w.services.SaveSettings = func(string, json.RawMessage, json.RawMessage) error { return nil }
	w.handle(snoofer.Request{ID: "aec.engine", Value: "localvqe-aec"})
	m.refuseNil = true
	w.step(ctx)
	if old.closed || w.engine != old || w.releaseErr == "" || !old.last().Bypass {
		t.Fatal("unconfirmed detach must retain bypassed engine")
	}
	m.refuseNil = false
	w.step(ctx)
	if w.engine != neural || !w.hooked || !old.closed || w.configured == false {
		t.Fatal("engine did not switch")
	}
	if control(t, w, "aec.strength").Available {
		t.Fatal("neural engine exposed inapplicable strength")
	}
	if control(t, w, "aec.engine").Value != "localvqe-aec" {
		t.Fatal("wrong selection")
	}
	if err := w.shutdown(); err != nil || !neural.closed {
		t.Fatal("neural cleanup", err)
	}
}

func TestNeuralLoadFailureCanSwitchBack(t *testing.T) {
	ctx := context.Background()
	m := &fakeMixer{targets: speakers}
	old := &fakeEngine{}
	w := newWorker(Settings{}, m, old)
	w.step(ctx)
	opens := 0
	replacement := &fakeEngine{}
	w.open = func(choice string) (canceller, error) {
		opens++
		if choice == "aec3" {
			return replacement, nil
		}
		return nil, errors.New("missing model")
	}
	w.handle(snoofer.Request{ID: "aec.engine", Value: "localvqe-voice"})
	w.step(ctx)
	w.step(ctx)
	if !old.closed || m.hook != nil || opens != 1 || control(t, w, "aec.status").Value != "Error" {
		t.Fatal("load failure retried or retained processing", opens)
	}
	w.handle(snoofer.Request{ID: "aec.engine", Value: "aec3"})
	w.step(ctx)
	if w.engine != replacement || !w.hooked || !control(t, w, "aec.strength").Available {
		t.Fatal("could not switch back")
	}
	for _, value := range []string{"aec3", "localvqe-aec", "localvqe-voice"} {
		if err := validate(snoofer.MarshalSettings(Settings{Engine: value})); err != nil {
			t.Fatal(err)
		}
	}
	if validate(json.RawMessage(`{"engine":"unknown"}`)) == nil {
		t.Fatal("unknown engine accepted")
	}
	_ = w.shutdown()
}
