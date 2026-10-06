// Package appaudio controls the Windows volume and mute of individual apps.
package appaudio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"sound-snoofer/internal/windowsaudio"
	"sound-snoofer/snoofer"
)

// Settings are the user's picks and rules. Volumes are never stored; Windows
// remembers them per app.
type Settings struct {
	Picked        []string `json:"picked,omitempty"`
	Rules         []Rule   `json:"rules,omitempty"`
	RecentMinutes int      `json:"recent_minutes,omitempty"` // Zero means 5.
}

func (s Settings) window() time.Duration {
	if s.RecentMinutes <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(s.RecentMinutes) * time.Minute
}

const (
	heardPeak   = 0.001 // −60 dBFS: quieter peaks do not count as playing.
	volumeStep  = 0.02  // Per dial detent.
	volumeMatch = 0.005 // Read-back tolerance.
)

type timing struct {
	poll, list, observe, retry time.Duration
}

var defaultTiming = timing{poll: 100 * time.Millisecond, list: time.Second, observe: 2 * time.Second, retry: 5 * time.Second}

// Plugin returns inert metadata; sessions are opened only when started.
func Plugin() snoofer.Plugin {
	return snoofer.Plugin{ID: "appaudio", Label: "App audio", Validate: validate, Defaults: snoofer.MarshalSettings(Settings{}),
		Start: func(ctx context.Context, s snoofer.Services, raw json.RawMessage, _ map[string]snoofer.Instance) (snoofer.Instance, error) {
			return start(ctx, s, raw, windowsaudio.OpenSessions, defaultTiming)
		}}
}

func validate(raw json.RawMessage) error {
	var s Settings
	if err := snoofer.DecodeSettings(raw, &s); err != nil {
		return err
	}
	if s.RecentMinutes < 0 || s.RecentMinutes > 24*60 {
		return fmt.Errorf("recent_minutes must be 0–1440")
	}
	_, err := compileRules(s.Rules)
	return err
}

type instance struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func (i *instance) Stop(ctx context.Context) error {
	i.cancel()
	select {
	case <-i.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// request is a write awaiting read-back.
type request struct {
	volume, mute bool // Which parts were written.
	wantVolume   float64
	wantMute     bool
	at           time.Time
	failed       string
}

type worker struct {
	services snoofer.Services
	raw      json.RawMessage
	settings Settings
	rules    []compiledRule
	timing   timing

	open       func() (windowsaudio.SessionBackend, error)
	backend    windowsaudio.SessionBackend
	backendErr string
	retryAt    time.Time
	listAt     time.Time

	apps    []*app
	peaks   map[string]float64   // Session key to linear peak.
	heard   map[string]time.Time // App key to last audible peak.
	pending map[string]request   // App key to the latest write.
	ignored map[string]bool      // App key: a write was not observed in time.
	editErr string
}

func start(ctx context.Context, s snoofer.Services, raw json.RawMessage, open func() (windowsaudio.SessionBackend, error), t timing) (snoofer.Instance, error) {
	if err := validate(raw); err != nil {
		return nil, err
	}
	var settings Settings
	if err := snoofer.DecodeSettings(raw, &settings); err != nil {
		return nil, err
	}
	rules, err := compileRules(settings.Rules)
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	i := &instance{cancel: cancel, done: make(chan struct{})}
	w := &worker{services: s, raw: raw, settings: settings, rules: rules, timing: t, open: open,
		peaks: map[string]float64{}, heard: map[string]time.Time{}, pending: map[string]request{}, ignored: map[string]bool{}}
	commands := make(chan snoofer.Request, 8)
	go func() {
		// COM objects belong to this OS thread for the worker's whole life.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(i.done)
		defer s.Controls.Remove("appaudio")
		defer w.closeBackend()
		ticker := time.NewTicker(t.poll)
		defer ticker.Stop()
		for {
			w.step(time.Now())
			w.publish(commands, time.Now())
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
			case r := <-commands:
				w.handle(r, time.Now())
			}
		}
	}()
	return i, nil
}

func (w *worker) closeBackend() {
	if w.backend != nil {
		w.backend.Close()
		w.backend = nil
	}
}

// step opens the backend when needed, rescans sessions on schedule and
// samples peaks.
func (w *worker) step(now time.Time) {
	if w.backend == nil {
		if now.Before(w.retryAt) {
			return
		}
		backend, err := w.open()
		if err != nil {
			w.backendErr = err.Error()
			w.retryAt = now.Add(w.timing.retry)
			return
		}
		w.backend, w.backendErr, w.listAt = backend, "", time.Time{}
	}
	if !now.Before(w.listAt) {
		sessions, err := w.backend.List()
		if err != nil {
			w.backendErr = err.Error()
			w.closeBackend()
			w.retryAt = now.Add(w.timing.retry)
			w.apps = nil
			return
		}
		w.backendErr = ""
		w.apps = group(sessions, w.rules)
		w.listAt = now.Add(w.timing.list)
		w.observe(now)
	}
	peaks := map[string]float64{}
	for _, a := range w.apps {
		for _, s := range a.Sessions {
			peak, err := w.backend.Peak(s.Key)
			if err != nil {
				continue
			}
			peaks[s.Key] = peak
			if peak > heardPeak {
				w.heard[key(a.Name)] = now
			}
		}
	}
	w.peaks = peaks
}

// observe settles writes that the latest scan confirms, and marks those
// still unconfirmed after the observe timeout.
func (w *worker) observe(now time.Time) {
	for k, r := range w.pending {
		a := w.find(k)
		if a == nil || len(a.Sessions) == 0 {
			delete(w.pending, k)
			continue
		}
		matched := true
		for _, s := range a.Sessions {
			if r.volume && math.Abs(s.Volume-r.wantVolume) > volumeMatch {
				matched = false
			}
			if r.mute && s.Muted != r.wantMute {
				matched = false
			}
		}
		switch {
		case matched && r.failed == "":
			delete(w.pending, k)
			delete(w.ignored, k)
		case now.Sub(r.at) > w.timing.observe:
			delete(w.pending, k)
			if r.failed == "" {
				w.ignored[k] = true
			}
		}
	}
}

func (w *worker) find(k string) *app {
	for _, a := range w.apps {
		if !a.Hidden && key(a.Name) == k {
			return a
		}
	}
	return nil
}

func (w *worker) handle(r snoofer.Request, now time.Time) {
	if r.ID == "appaudio.edit" {
		w.edit(r.Value)
		return
	}
	var target *app
	for _, a := range w.apps {
		if !a.Hidden && controlID(a.Name) == r.ID {
			target = a
		}
	}
	if target == nil || len(target.Sessions) == 0 || !w.services.Live || w.backend == nil {
		return // Closed apps and preview ignore input.
	}
	k := key(target.Name)
	current, writing := w.pending[k]
	next := request{at: now}
	switch r.Operation {
	case "press":
		next.mute = true
		next.wantMute = target.muteState() != "all"
		if writing && current.mute {
			next.wantMute = !current.wantMute
		}
	case "adjust":
		base := target.volume()
		if writing && current.volume {
			base = current.wantVolume
		}
		next.volume = true
		next.wantVolume = math.Round(min(1, max(0, base+float64(r.Delta)*volumeStep))*100) / 100
	case "set":
		percent, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(r.Value), "%"), 64)
		if err != nil || percent < 0 || percent > 100 {
			return
		}
		next.volume = true
		next.wantVolume = math.Round(percent) / 100
	default:
		return
	}
	var failures []error
	for _, s := range target.Sessions {
		if next.volume {
			failures = append(failures, w.backend.SetVolume(s.Key, next.wantVolume))
		}
		if next.mute {
			failures = append(failures, w.backend.SetMute(s.Key, next.wantMute))
		}
	}
	if err := errors.Join(failures...); err != nil {
		next.failed = err.Error()
	}
	delete(w.ignored, k)
	w.pending[k] = next
	w.listAt = now // Read back on the next step.
}

// appValue is the shown value: the requested one while a write is pending.
func (w *worker) appValue(a *app) (string, string) {
	if len(a.Sessions) == 0 {
		return "Closed", ""
	}
	k := key(a.Name)
	volume, mute := a.volume(), a.muteState()
	status := ""
	if r, ok := w.pending[k]; ok {
		status = "Pending"
		if r.failed != "" {
			status = r.failed
		}
		if r.volume {
			volume = r.wantVolume
		}
		if r.mute {
			mute = "none"
			if r.wantMute {
				mute = "all"
			}
		}
	} else if w.ignored[k] {
		status = "Ignored by app"
	}
	switch mute {
	case "all":
		return "Muted", status
	case "some":
		return "Mixed", status
	}
	return fmt.Sprintf("%d%%", int(math.Round(volume*100))), status
}

type appView struct {
	ID, Name, Rule       string
	Picked, Hidden, Open bool
	Executables, Devices []string
	PIDs                 []uint32
	Sessions             int
	LastHeard            *time.Time `json:",omitempty"` // Minute precision, so state does not churn while audio plays.
}

type statusView struct {
	Apps          []appView
	RecentMinutes int
}

func (w *worker) publish(commands chan snoofer.Request, now time.Time) {
	shown := visible(w.apps, w.settings.Picked, w.heard, w.settings.window(), now)
	var controls []snoofer.Control
	for n, a := range shown {
		value, status := w.appValue(a)
		meter := snoofer.Meter{Present: true, At: now}
		if len(a.Sessions) > 0 {
			peak := 0.0
			for _, s := range a.Sessions {
				peak = max(peak, w.peaks[s.Key])
			}
			meter.Known = true
			meter.DB = max(-120, 20*math.Log10(max(peak, 1e-6)))
		}
		controls = append(controls, snoofer.Control{ID: controlID(a.Name), Label: a.Name, ShortLabel: a.Name, Group: "App audio",
			Collection: "appaudio.apps", CollectionLabel: "Apps", Order: n + 1, Kind: "numeric", Icon: "app-audio", Artwork: a.Icon,
			Value: value, Status: status, Meter: meter, Operations: []string{"press", "adjust", "set"}, Available: true})
	}
	view := statusView{RecentMinutes: int(w.settings.window() / time.Minute)}
	listed := map[string]bool{}
	for _, a := range append(append([]*app{}, w.apps...), shown...) {
		k := key(a.Name)
		if a.Hidden {
			k = "hidden\x00" + k
		}
		if listed[k] {
			continue
		}
		listed[k] = true
		v := appView{ID: controlID(a.Name), Name: a.Name, Rule: a.Rule, Hidden: a.Hidden, Open: len(a.Sessions) > 0, Sessions: len(a.Sessions),
			Picked: slices.ContainsFunc(w.settings.Picked, func(p string) bool { return key(p) == key(a.Name) })}
		for _, s := range a.Sessions {
			if s.Path != "" && !slices.Contains(v.Executables, s.Path) {
				v.Executables = append(v.Executables, s.Path)
			}
			if !slices.Contains(v.Devices, s.Device) {
				v.Devices = append(v.Devices, s.Device)
			}
			if s.PID != 0 && !slices.Contains(v.PIDs, s.PID) {
				v.PIDs = append(v.PIDs, s.PID)
			}
		}
		if t, ok := w.heard[key(a.Name)]; ok && !a.Hidden {
			minute := t.Truncate(time.Minute)
			v.LastHeard = &minute
		}
		view.Apps = append(view.Apps, v)
	}
	data, _ := json.Marshal(view) // Plain fields always marshal.
	value := fmt.Sprintf("%d apps", len(shown))
	switch {
	case w.backendErr != "":
		value = "Unavailable"
	case !w.services.Live:
		value = "Preview"
	}
	controls = append(controls,
		snoofer.Control{ID: "appaudio.status", Label: "App audio", Group: "App audio", Kind: "status", Value: value, Status: firstNonEmpty(w.backendErr, w.editErr), ViewData: data, Available: true},
		snoofer.Control{ID: "appaudio.edit", Label: "App audio edit", Group: "App audio", Kind: "text", Operations: []string{"set"}, Available: true})
	_ = w.services.Controls.Publish("appaudio", controls, func(ctx context.Context, r snoofer.Request) error {
		select {
		case commands <- r:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf("app audio busy")
		}
	})
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
