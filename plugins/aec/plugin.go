// Package aec cancels speaker echo from the managed microphone with WebRTC
// AEC3, running inside Voicemeeter's audio callback through the audio
// plugin's insert hook.
package aec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	engine "sound-snoofer/internal/aec"
	"sound-snoofer/internal/voicemeeter"
	"sound-snoofer/plugins/audio"
	"sound-snoofer/snoofer"
)

// Settings are the saved echo cancellation preferences.
type Settings struct {
	Engine   string `json:"engine,omitempty"`   // Empty defaults to WebRTC AEC3.
	Mode     string `json:"mode,omitempty"`     // "auto" (default), "on" or "off".
	Strength string `json:"strength,omitempty"` // "strong" (default), "balanced" or "gentle".
	// SpeakerOutputs are Go regexps naming speaker playback devices for Auto.
	// Empty means any device whose name does not look like headphones.
	SpeakerOutputs []string `json:"speaker_outputs,omitempty"`
}

var (
	engines      = []string{"aec3", "localvqe-aec", "localvqe-voice"}
	engineLabels = map[string]string{"aec3": "WebRTC AEC3", "localvqe-aec": "LocalVQE echo-only", "localvqe-voice": "LocalVQE voice cleanup"}
	modes        = []string{"auto", "on", "off"}
	strengths    = []string{"strong", "balanced", "gentle"}
	modeLabels   = map[string]string{"auto": "Auto", "on": "On", "off": "Off"}
	strengthOf   = map[string]engine.Strength{"strong": engine.Strong, "balanced": engine.Balanced, "gentle": engine.Gentle}
	strengthText = map[string]string{"strong": "Strong", "balanced": "Balanced", "gentle": "Gentle"}
	headphones   = regexp.MustCompile(`(?i)head(phone|set)|ear(phone|bud)|airpods|\bbuds\b`)
)

func (s Settings) engine() string {
	if s.Engine == "" {
		return "aec3"
	}
	return s.Engine
}

func (s Settings) mode() string {
	if s.Mode == "" {
		return "auto"
	}
	return s.Mode
}

func (s Settings) strength() string {
	if s.Strength == "" {
		return "strong"
	}
	return s.Strength
}

func (s Settings) speakers() ([]*regexp.Regexp, error) {
	var out []*regexp.Regexp
	for _, pattern := range s.SpeakerOutputs {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("speaker_outputs %q: %w", pattern, err)
		}
		out = append(out, re)
	}
	return out, nil
}

func validate(raw json.RawMessage) error {
	var s Settings
	if err := snoofer.DecodeSettings(raw, &s); err != nil {
		return err
	}
	if !slices.Contains(engines, s.engine()) {
		return fmt.Errorf("unknown echo engine %q", s.Engine)
	}
	if !slices.Contains(modes, s.mode()) {
		return fmt.Errorf("mode must be auto, on or off")
	}
	if !slices.Contains(strengths, s.strength()) {
		return fmt.Errorf("strength must be strong, balanced or gentle")
	}
	_, err := s.speakers()
	return err
}

// canceller is the native engine as the plugin uses it.
type canceller interface {
	Configure(engine.Config) error
	Stats() (engine.Stats, error)
	Reset() error
	Hook() voicemeeter.InsertHook
	Close() error
}

// mixer is the audio plugin's echo API.
type mixer interface {
	EchoTargets() audio.EchoTargets
	SetEchoInsert(context.Context, *voicemeeter.InsertHook) error
}

// Plugin returns inert metadata; the engine loads only when first needed.
func Plugin() snoofer.Plugin {
	return snoofer.Plugin{ID: "aec", Label: "Echo cancellation", Validate: validate, Defaults: snoofer.MarshalSettings(Settings{}), Requires: []string{"audio"},
		Start: func(ctx context.Context, s snoofer.Services, raw json.RawMessage, deps map[string]snoofer.Instance) (snoofer.Instance, error) {
			return start(ctx, s, raw, deps["audio"].(*audio.Instance), openBesideExecutable, time.Second)
		}}
}

func openBesideExecutable(choice string) (canceller, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(exe)
	if choice != "aec3" {
		model := "localvqe-v1.4-aec-200K-f32.gguf"
		if choice == "localvqe-voice" {
			model = "localvqe-v1.3-4.8M-f32.gguf"
		}
		path := filepath.Join(dir, "models", model)
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("neural model unavailable: %w", err)
		}
		e, err := engine.OpenNeural(filepath.Join(dir, "snoofer-neural-aec.dll"), path)
		if err != nil {
			return nil, fmt.Errorf("%s failed to load; mic passes through: %w", engineLabels[choice], err)
		}
		return e, nil
	}
	e, err := engine.Open(filepath.Join(dir, "snoofer-aec.dll"))
	if err != nil {
		return nil, fmt.Errorf("echo cancellation engine missing or unloadable beside executable: %w", err)
	}
	return e, nil
}

type instance struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error // Cleanup outcome, written before done closes.
}

// Stop removes the hook and frees the engine, unless removal is unconfirmed:
// then the engine stays loaded until the process exits.
func (i *instance) Stop(ctx context.Context) error {
	i.cancel()
	select {
	case <-i.done:
		return i.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// releaseTimeout bounds waiting for the audio worker to drop the hook.
const releaseTimeout = 5 * time.Second

type worker struct {
	services snoofer.Services
	mixer    mixer
	open     func(string) (canceller, error)
	raw      json.RawMessage
	settings Settings
	speakers []*regexp.Regexp
	requests chan snoofer.Request

	loaded     string
	attempted  string
	loading    bool
	engine     canceller
	engineErr  string
	hooked     bool   // A hook was requested and not confirmed removed.
	armed      uint32 // The audio stream the engine was last armed on.
	applied    engine.Config
	configured bool
	releaseErr string
	saveErr    string
	stats      engine.Stats
	statsErr   string
	active     bool   // Wanted processing on this step.
	reason     string // Why not active.
}

func start(ctx context.Context, s snoofer.Services, raw json.RawMessage, m mixer, open func(string) (canceller, error), interval time.Duration) (snoofer.Instance, error) {
	var settings Settings
	if err := snoofer.DecodeSettings(raw, &settings); err != nil {
		return nil, err
	}
	speakers, err := settings.speakers()
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	i := &instance{cancel: cancel, done: make(chan struct{})}
	w := &worker{services: s, mixer: m, open: open, raw: append(json.RawMessage(nil), raw...), settings: settings, speakers: speakers, requests: make(chan snoofer.Request, 8)}
	go func() {
		defer close(i.done)
		defer s.Controls.Remove("aec")
		i.err = w.run(runCtx, interval)
	}()
	return i, nil
}

func (w *worker) run(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		w.step(ctx)
		w.publish()
		select {
		case <-ctx.Done():
			return w.shutdown()
		case r := <-w.requests:
			w.handle(r)
		case <-ticker.C:
		}
	}
}

// decide reports whether echo cancellation should process now, or why not.
func decide(s Settings, speakers []*regexp.Regexp, t audio.EchoTargets) (bool, string) {
	switch {
	case s.mode() == "off":
		return false, "Off"
	case t.Reason != "":
		return false, t.Reason
	case s.mode() == "auto" && !isSpeaker(t.Playback, speakers):
		return false, "Headphones"
	}
	return true, ""
}

func isSpeaker(device string, speakers []*regexp.Regexp) bool {
	if len(speakers) == 0 {
		return !headphones.MatchString(device)
	}
	for _, re := range speakers {
		if re.MatchString(device) {
			return true
		}
	}
	return false
}

func (w *worker) step(ctx context.Context) {
	if w.engine != nil && w.loaded != w.settings.engine() {
		if w.hooked {
			w.deactivate(ctx)
		}
		if w.hooked {
			return
		}
		closeErr := w.engine.Close()
		w.engine = nil
		w.configured = false
		w.stats = engine.Stats{}
		w.statsErr = ""
		if closeErr != nil {
			w.engineErr = closeErr.Error()
			w.attempted = w.settings.engine()
			return
		}
	}
	if w.attempted != w.settings.engine() {
		w.engineErr = ""
	}
	targets := w.mixer.EchoTargets()
	w.active, w.reason = decide(w.settings, w.speakers, targets)
	if w.active {
		w.activate(ctx, targets)
	} else if w.hooked {
		w.deactivate(ctx)
	}
	w.statsErr = ""
	if w.engine != nil {
		stats, err := w.engine.Stats()
		w.stats = stats
		if err != nil {
			w.statsErr = err.Error()
		}
	}
	w.recover(targets.Stream)
}

func (w *worker) activate(ctx context.Context, targets audio.EchoTargets) {
	if w.engine == nil {
		if w.attempted == w.settings.engine() && w.engineErr != "" {
			return
		}
		w.loading = true
		w.publish()
		w.attempted = w.settings.engine()
		e, err := w.open(w.settings.engine())
		w.loading = false
		if err != nil {
			w.engineErr = err.Error()
			return
		}
		w.engine, w.engineErr = e, ""
		w.loaded = w.settings.engine()
	}
	// A failed installation may still own a queued hook. Remove it before retrying.
	if w.hooked && w.engineErr != "" {
		w.deactivate(ctx)
		if w.hooked {
			return
		}
	}
	config := engine.Config{Mic: targets.Mic, Reference: targets.Reference, Strength: strengthOf[w.settings.strength()]}
	if !w.configured || config != w.applied {
		if err := w.engine.Configure(config); err != nil {
			w.engineErr = err.Error()
			return
		}
		w.applied, w.configured = config, true
	}
	if !w.hooked {
		hook := w.engine.Hook()
		w.hooked = true // Keep ownership until removal is acknowledged, even if installation fails.
		if err := w.mixer.SetEchoInsert(ctx, &hook); err != nil {
			w.engineErr = err.Error()
			return
		}
		w.armed = targets.Stream
	}
	w.engineErr = ""
}

// recover resets a failed engine once per new audio stream. A stream that ends
// between the inserts fails the next stream's first callback, possibly before
// the new stream count is published; a failure on the armed stream stays latched.
func (w *worker) recover(stream uint32) {
	if !w.stats.Failed || !w.hooked || stream == w.armed {
		return
	}
	if err := w.engine.Reset(); err != nil {
		w.engineErr = err.Error()
		return
	}
	w.armed = stream
}

// deactivate bypasses the engine at once, then removes the hook.
func (w *worker) deactivate(ctx context.Context) {
	bypass := w.applied
	bypass.Bypass = true
	if err := w.engine.Configure(bypass); err == nil {
		w.applied = bypass
	}
	releaseCtx, cancel := context.WithTimeout(ctx, releaseTimeout)
	defer cancel()
	if err := w.mixer.SetEchoInsert(releaseCtx, nil); err != nil {
		w.releaseErr = err.Error()
		return
	}
	w.hooked, w.releaseErr = false, ""
}

// shutdown removes the hook with a fresh deadline (the run context is
// already cancelled) and frees the engine only once removal is confirmed.
func (w *worker) shutdown() error {
	if w.engine == nil {
		return nil
	}
	if w.hooked {
		ctx, cancel := context.WithTimeout(context.Background(), releaseTimeout)
		defer cancel()
		if err := w.mixer.SetEchoInsert(ctx, nil); err != nil {
			return fmt.Errorf("echo cancellation engine kept loaded: %w", err)
		}
	}
	return w.engine.Close()
}

func (w *worker) handle(r snoofer.Request) {
	next := w.settings
	switch r.ID {
	case "aec.engine":
		if !slices.Contains(engines, r.Value) {
			w.saveErr = "unknown engine " + r.Value
			return
		}
		next.Engine = r.Value
	case "aec.mode":
		if !slices.Contains(modes, r.Value) {
			w.saveErr = "unknown mode " + r.Value
			return
		}
		next.Mode = r.Value
	case "aec.strength":
		if !slices.Contains(strengths, r.Value) {
			w.saveErr = "unknown strength " + r.Value
			return
		}
		next.Strength = r.Value
	default:
		return
	}
	w.saveErr = ""
	if err := w.save(next); err != nil {
		w.saveErr = err.Error()
	} else if r.ID == "aec.engine" {
		w.attempted = ""
	}
}

func (w *worker) save(next Settings) error {
	if w.services.SaveSettings == nil {
		return errors.New("settings cannot be saved")
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err := w.services.SaveSettings("aec", w.raw, raw); err != nil {
		return err
	}
	w.raw, w.settings = raw, next
	return nil
}

// status is the state word and its detail line.
func (w *worker) status() (string, string) {
	if w.loading {
		return "Wait", "Loading " + engineLabels[w.settings.engine()]
	}
	failure := firstNonEmpty(w.releaseErr, w.engineErr, w.statsErr)
	switch {
	case w.stats.Failed:
		if w.stats.Reason != "" {
			return "Error", "Engine error (" + w.stats.Reason + "); mic passes through"
		}
		return "Error", "Engine error; mic passes through"
	case failure != "":
		return "Error", failure
	case !w.active && w.settings.mode() == "off":
		return "Off", ""
	case !w.active:
		return "Idle", w.reason
	case w.stats.SampleRate != 0 && !engine.Supported(w.stats.SampleRate):
		return "Idle", "Needs 48 kHz"
	case !w.stats.Active:
		return "Wait", ""
	}
	detail := []string{}
	if w.settings.engine() != "aec3" {
		detail = append(detail, "16 kHz mono")
		if w.stats.LatencyMs > 0 {
			detail = append(detail, fmt.Sprintf("~%d ms processing latency", w.stats.LatencyMs))
		}
	}
	if w.stats.ERLEKnown {
		detail = append(detail, fmt.Sprintf("−%.0f dB echo", w.stats.ERLE))
	}
	if w.stats.DelayKnown {
		detail = append(detail, fmt.Sprintf("%d ms", w.stats.DelayMs))
	}
	return "Active", strings.Join(detail, " · ")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (w *worker) publish() {
	value, detail := w.status()
	canSave := w.services.SaveSettings != nil
	controls := []snoofer.Control{
		{ID: "aec.engine", Label: "Echo engine", ShortLabel: "Engine", Group: "Echo cancellation", Kind: "selection", Icon: "echo", Value: w.settings.engine(), Options: engines, OptionLabels: engineLabels, Status: w.saveErr, Operations: []string{"set"}, Available: canSave},
		{ID: "aec.mode", Label: "Echo cancellation", ShortLabel: "Echo", Group: "Echo cancellation", Kind: "selection", Icon: "echo",
			Value: w.settings.mode(), Options: modes, OptionLabels: modeLabels, Status: w.saveErr, Operations: []string{"set"}, Available: canSave},
		{ID: "aec.strength", Label: "Echo strength", ShortLabel: "Strength", Group: "Echo cancellation", Kind: "selection", Icon: "echo",
			Value: w.settings.strength(), Options: strengths, OptionLabels: strengthText, Operations: []string{"set"}, Available: canSave && w.settings.engine() == "aec3"},
		{ID: "aec.status", Label: "Echo cancellation status", ShortLabel: "Echo", Group: "Echo cancellation", Kind: "status", Icon: "echo",
			Value: value, Status: detail, Available: true},
	}
	requests := w.requests
	_ = w.services.Controls.Publish("aec", controls, func(ctx context.Context, r snoofer.Request) error {
		select {
		case requests <- r:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf("echo cancellation queue full")
		}
	})
}
