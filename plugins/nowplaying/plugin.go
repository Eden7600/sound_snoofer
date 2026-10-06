// Package nowplaying shows and controls media sessions: every Windows
// player, and each browser tab through the Snoofer browser extension.
package nowplaying

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"sound-snoofer/internal/mediasessions"
	"sound-snoofer/snoofer"
)

// Settings configure the browser bridge.
type Settings struct {
	Port  int    `json:"port,omitempty"`  // Zero means 47815.
	Token string `json:"token,omitempty"` // Created on first start.
}

const defaultPort = 47815

func (s Settings) port() int {
	if s.Port == 0 {
		return defaultPort
	}
	return s.Port
}

const (
	seekStep    = 5000 // Milliseconds per dial detent.
	focusHold   = 30 * time.Second
	seekMatchMs = 3000
)

type timing struct {
	poll, windows, observe, retry time.Duration
}

var defaultTiming = timing{poll: 100 * time.Millisecond, windows: 500 * time.Millisecond, observe: 3 * time.Second, retry: 5 * time.Second}

// windowsSource is the Windows media sessions companion.
type windowsSource interface {
	Snapshot() ([]mediasessions.Session, error)
	Art(id string) ([]byte, error)
	Command(id string, op mediasessions.Op, value int64) error
	Close() error
}

// Plugin returns inert metadata; nothing is loaded until started.
func Plugin() snoofer.Plugin {
	return snoofer.Plugin{ID: "nowplaying", Label: "Now playing", Validate: validate, Defaults: snoofer.MarshalSettings(Settings{}),
		Start: func(ctx context.Context, s snoofer.Services, raw json.RawMessage, _ map[string]snoofer.Instance) (snoofer.Instance, error) {
			exe, err := os.Executable()
			if err != nil {
				return nil, err
			}
			library := filepath.Join(filepath.Dir(exe), "snoofer-media.dll")
			open := func() (windowsSource, error) { return mediasessions.Open(library) }
			return start(ctx, s, raw, open, defaultTiming)
		}}
}

func validate(raw json.RawMessage) error {
	var s Settings
	if err := snoofer.DecodeSettings(raw, &s); err != nil {
		return err
	}
	if s.Port != 0 && (s.Port < 1024 || s.Port > 65535) {
		return fmt.Errorf("port must be 1024–65535")
	}
	return nil
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

// pendingCommand is a request awaiting an observed change.
type pendingCommand struct {
	op          string
	wantPlaying bool
	wantMuted   bool
	wantMs      int64
	at          time.Time
}

type worker struct {
	services  snoofer.Services
	timing    timing
	raw       json.RawMessage
	settings  Settings
	bridge    *bridge // Nil when the port could not be opened.
	bridgeErr string

	openWindows func() (windowsSource, error)
	win         windowsSource
	winErr      string
	retryAt     time.Time
	pollAt      time.Time
	winSessions []mediasessions.Session
	winArt      map[string]string // ID and ArtKey to thumbnail.

	browsers   map[string]browserUpdate
	browserArt map[string]string // Session key and ArtKey to thumbnail.
	send       func(browser string, c browserCommand) error

	sessions []session
	started  map[string]time.Time // When each session last began playing.
	playing  map[string]bool
	focus    string
	pressed  time.Time // The last time the user chose what to control.
	pending  map[string]pendingCommand
	failure  map[string]string
}

func start(ctx context.Context, s snoofer.Services, raw json.RawMessage, open func() (windowsSource, error), t timing) (snoofer.Instance, error) {
	if err := validate(raw); err != nil {
		return nil, err
	}
	var settings Settings
	if err := snoofer.DecodeSettings(raw, &settings); err != nil {
		return nil, err
	}
	w := newWorker(s, open, t)
	w.raw, w.settings = raw, settings
	if settings.Token == "" {
		if err := w.resetToken(); err != nil {
			return nil, fmt.Errorf("create browser bridge token: %w", err)
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	i := &instance{cancel: cancel, done: make(chan struct{})}
	updates := make(chan browserUpdate, 16)
	commands := make(chan snoofer.Request, 8)
	// A busy port leaves Windows sessions working; the GUI shows why tabs are missing.
	if b, err := listen(runCtx, w.settings.port(), w.settings.Token, updates); err != nil {
		w.bridgeErr = err.Error()
	} else {
		w.bridge, w.send = b, b.send
	}
	go func() {
		if w.bridge != nil {
			defer w.bridge.close()
		}
		// The Windows companion belongs to this OS thread for its whole life.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(i.done)
		defer s.Controls.Remove("nowplaying")
		defer w.closeWindows()
		ticker := time.NewTicker(t.poll)
		defer ticker.Stop()
		for {
			now := time.Now()
			w.step(now)
			w.publish(commands, now)
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
			case u := <-updates:
				w.browser(u)
			case r := <-commands:
				w.handle(r, time.Now())
			}
		}
	}()
	return i, nil
}

func newWorker(s snoofer.Services, open func() (windowsSource, error), t timing) *worker {
	return &worker{services: s, timing: t, openWindows: open, winArt: map[string]string{}, browsers: map[string]browserUpdate{}, browserArt: map[string]string{},
		send:    func(string, browserCommand) error { return errors.New("browser bridge unavailable") },
		started: map[string]time.Time{}, playing: map[string]bool{}, pending: map[string]pendingCommand{}, failure: map[string]string{}}
}

func (w *worker) closeWindows() {
	if w.win != nil {
		_ = w.win.Close() // Shutdown: nothing useful can follow a failed release.
		w.win = nil
	}
}

// step refreshes Windows sessions on schedule, then merges both sources and
// settles focus and pending commands.
func (w *worker) step(now time.Time) {
	if w.win == nil && !now.Before(w.retryAt) {
		win, err := w.openWindows()
		if err != nil {
			w.winErr, w.retryAt = err.Error(), now.Add(w.timing.retry)
		} else {
			w.win, w.winErr, w.pollAt = win, "", time.Time{}
		}
	}
	if w.win != nil && !now.Before(w.pollAt) {
		w.pollAt = now.Add(w.timing.windows)
		sessions, err := w.win.Snapshot()
		if err != nil {
			w.winErr = err.Error()
			w.closeWindows()
			w.retryAt = now.Add(w.timing.retry)
			w.winSessions = nil
		} else {
			w.winErr, w.winSessions = "", sessions
			w.windowsArt()
		}
	}
	w.merge(now)
}

// windowsArt fetches artwork only for tracks not seen before, and forgets
// artwork of tracks no longer shown.
func (w *worker) windowsArt() {
	keep := map[string]string{}
	for _, s := range w.winSessions {
		if s.ArtKey == "" {
			continue
		}
		k := s.ID + "\x00" + s.ArtKey
		art, ok := w.winArt[k]
		if !ok {
			if data, err := w.win.Art(s.ID); err == nil && data != nil {
				art, _ = thumbnail(data) // Unreadable artwork falls back to the play symbol.
			}
		}
		keep[k] = art
	}
	w.winArt = keep
}

// browser applies one browser's report. Artwork arrives only when it changes.
func (w *worker) browser(u browserUpdate) {
	if !u.Connected {
		delete(w.browsers, u.Browser)
		return
	}
	for _, s := range u.Sessions {
		if s.Art != "" && validArtwork(s.Art) {
			w.browserArt[u.Browser+":"+s.ID+"\x00"+s.ArtKey] = s.Art
		}
	}
	w.browsers[u.Browser] = u
}

func (w *worker) merge(now time.Time) {
	connected := map[string]bool{}
	for name := range w.browsers {
		connected[name] = true
	}
	var all []session
	for _, s := range w.winSessions {
		if s.Status == "closed" || shadowed(s.App, connected) {
			continue
		}
		all = append(all, fromWindows(s, w.winArt[s.ID+"\x00"+s.ArtKey]))
	}
	browserNames := make([]string, 0, len(w.browsers))
	for name := range w.browsers {
		browserNames = append(browserNames, name)
	}
	slices.Sort(browserNames)
	usedArt := map[string]bool{}
	for _, name := range browserNames {
		for _, s := range w.browsers[name].Sessions {
			k := name + ":" + s.ID + "\x00" + s.ArtKey
			usedArt[k] = true
			all = append(all, fromBrowser(name, s, w.browserArt[k]))
		}
	}
	for k := range w.browserArt {
		if !usedArt[k] {
			delete(w.browserArt, k)
		}
	}
	// Focus moves to a session that has just started playing, unless the
	// user chose a session recently.
	present := map[string]bool{}
	for _, s := range all {
		present[s.Key] = true
		if s.playing() && !w.playing[s.Key] {
			w.started[s.Key] = now
			if now.Sub(w.pressed) > focusHold {
				w.focus = s.Key
			}
		}
		w.playing[s.Key] = s.playing()
	}
	for k := range w.playing {
		if !present[k] {
			delete(w.playing, k)
			delete(w.started, k)
			delete(w.failure, k)
		}
	}
	slices.SortStableFunc(all, func(a, b session) int { return w.started[b.Key].Compare(w.started[a.Key]) })
	if !present[w.focus] {
		w.focus = ""
		for _, s := range all {
			if s.playing() {
				w.focus = s.Key
				break
			}
		}
		if w.focus == "" && len(all) > 0 {
			w.focus = all[0].Key
		}
	}
	w.sessions = all
	w.observe(now)
}

// observe settles pending commands that the sources confirm, and reports
// those that are not confirmed in time.
func (w *worker) observe(now time.Time) {
	for k, p := range w.pending {
		s, ok := w.find(k)
		if !ok {
			delete(w.pending, k)
			continue
		}
		done := false
		switch p.op {
		case "toggle":
			done = s.playing() == p.wantPlaying
		case "seek":
			d := s.position(now) - p.wantMs
			done = d > -seekMatchMs && d < seekMatchMs
		case "mute":
			done = s.Muted == p.wantMuted
		}
		if done {
			delete(w.pending, k)
			delete(w.failure, k)
		} else if now.Sub(p.at) > w.timing.observe {
			delete(w.pending, k)
			w.failure[k] = "No response"
		}
	}
}

func (w *worker) find(key string) (session, bool) {
	for _, s := range w.sessions {
		if s.Key == key {
			return s, true
		}
	}
	return session{}, false
}

func (w *worker) handle(r snoofer.Request, now time.Time) {
	focused, ok := w.find(w.focus)
	switch r.ID {
	case "nowplaying.token-reset":
		w.bridgeErr = ""
		if err := w.resetToken(); err != nil {
			w.bridgeErr = err.Error()
		}
		return
	case "nowplaying.focus":
		for _, s := range w.sessions {
			if controlID(s.Key) == r.Value {
				w.focus, w.pressed = s.Key, now
			}
		}
		return
	case "nowplaying.dial":
		if !ok {
			return
		}
		w.pressed = now
		if r.Operation == "adjust" {
			target := focused.position(now) + int64(r.Delta)*seekStep
			if focused.DurationMs > 0 {
				target = min(target, focused.DurationMs-1000)
			}
			w.command(focused, "seek", max(0, target), now)
			return
		}
		w.command(focused, "toggle", 0, now)
		return
	case "nowplaying.toggle", "nowplaying.next", "nowplaying.prev", "nowplaying.mute":
		if ok {
			w.pressed = now
			w.command(focused, strings.TrimPrefix(r.ID, "nowplaying."), 0, now)
		}
		return
	}
	for _, s := range w.sessions {
		if controlID(s.Key) == r.ID {
			w.focus, w.pressed = s.Key, now
			w.command(s, "toggle", 0, now)
			return
		}
	}
}

// resetToken saves a new bridge token and drops connections that used the
// old one; the extension must be saved and reloaded to reconnect.
func (w *worker) resetToken() error {
	token, err := newToken()
	if err != nil {
		return err
	}
	next := w.settings
	next.Token = token
	raw, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if w.services.SaveSettings != nil {
		if err := w.services.SaveSettings("nowplaying", w.raw, raw); err != nil {
			return err
		}
	}
	w.raw, w.settings = raw, next
	if w.bridge != nil {
		w.bridge.setToken(token)
	}
	return nil
}

// command sends op to a session's source and records what to observe.
func (w *worker) command(s session, op string, value int64, now time.Time) {
	if !w.services.Live {
		return
	}
	var err error
	if s.Source == "windows" {
		if w.win == nil {
			err = errors.New("Windows media sessions unavailable")
		} else {
			native := map[string]mediasessions.Op{"toggle": mediasessions.Toggle, "next": mediasessions.Next, "prev": mediasessions.Previous, "seek": mediasessions.Seek}[op]
			if native == 0 {
				err = fmt.Errorf("%s is not supported here", op)
			} else {
				err = w.win.Command(s.ID, native, value)
			}
		}
	} else {
		err = w.send(s.Source, browserCommand{Type: "command", ID: s.ID, Op: op, Value: value})
	}
	if errors.Is(err, mediasessions.ErrDeclined) {
		err = errors.New("Declined")
	}
	if err != nil {
		w.failure[s.Key] = err.Error()
		delete(w.pending, s.Key)
		return
	}
	delete(w.failure, s.Key)
	switch op {
	case "toggle":
		w.pending[s.Key] = pendingCommand{op: op, wantPlaying: !s.playing(), at: now}
	case "seek":
		w.pending[s.Key] = pendingCommand{op: op, wantMs: value, at: now}
	case "mute":
		w.pending[s.Key] = pendingCommand{op: op, wantMuted: !s.Muted, at: now}
	}
}
