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
	"strconv"
	"strings"
	"time"

	"sound-snoofer/internal/mediasessions"
	"sound-snoofer/internal/placeholder"
	"sound-snoofer/snoofer"
)

// Settings configure the browser bridge.
type Settings struct {
	Port  int    `json:"port,omitempty"`  // Zero means 47815.
	Token string `json:"token,omitempty"` // Legacy from the first build; ignored.
	// DeckMediaOff hides media from the deck (sessions, the media dial and
	// transport) without changing the GUI.
	DeckMediaOff bool `json:"deck_media_off,omitempty"`
}

const defaultPort = 47815

func (s Settings) port() int {
	if s.Port == 0 {
		return defaultPort
	}
	return s.Port
}

const (
	seekStep    = 5000                   // Milliseconds per dial detent.
	scrubSettle = 250 * time.Millisecond // Quiet time after the last detent before seeking.
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

// scrub is a dial seek being gathered: detents move the target, and one seek
// is sent once the dial rests.
type scrub struct {
	key      string
	targetMs int64
	last     time.Time
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
	settings  Settings
	raw       json.RawMessage // The saved settings, for compare-and-swap saves.
	saveErr   string
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
	chosen   bool // The user chose focus by holding a session key; it sticks until reset.
	pending  map[string]pendingCommand
	failure  map[string]string
	scrub    *scrub            // The dial seek being gathered, if any.
	tiles    map[string]string // Player or site to placeholder artwork.
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
	w.settings, w.raw = settings, raw
	runCtx, cancel := context.WithCancel(ctx)
	i := &instance{cancel: cancel, done: make(chan struct{})}
	updates := make(chan browserUpdate, 16)
	commands := make(chan snoofer.Request, 8)
	// A busy port leaves Windows sessions working; the GUI shows why tabs are missing.
	if b, err := listen(runCtx, w.settings.port(), updates); err != nil {
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
		started: map[string]time.Time{}, playing: map[string]bool{}, pending: map[string]pendingCommand{}, failure: map[string]string{}, tiles: map[string]string{}}
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
	if w.scrub != nil && now.Sub(w.scrub.last) >= scrubSettle {
		target := *w.scrub
		w.scrub = nil
		if s, ok := w.find(target.key); ok {
			w.command(s, "seek", target.targetMs, now)
		}
	}
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
	tabTitles := map[string]bool{}
	for _, u := range w.browsers {
		for _, s := range u.Sessions {
			if s.Title != "" {
				tabTitles[s.Title] = true
			}
		}
	}
	var all []session
	for _, s := range w.winSessions {
		// A browser's own Windows session repeats one of its tabs. Hide it only
		// when that tab is reported, so it never vanishes from both sources.
		if s.Status == "closed" || (s.Title != "" && tabTitles[s.Title]) {
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
	present := map[string]bool{}
	for _, s := range all {
		present[s.Key] = true
		if s.playing() && !w.playing[s.Key] {
			w.started[s.Key] = now
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
	// A chosen focus sticks until reset or until its session ends. Otherwise
	// focus follows the most recently started session that is playing.
	if !present[w.focus] {
		w.chosen = false
	}
	if !w.chosen {
		for _, s := range all {
			if s.playing() {
				w.focus = s.Key
				break
			}
		}
	}
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
			if !done && now.Sub(p.at) > w.timing.observe {
				// Players report positions late or coarsely; show theirs
				// rather than an error.
				delete(w.pending, k)
				continue
			}
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

// artwork is the session's cover, or a placeholder in its player's or
// site's own colour.
func (w *worker) artwork(s session) string {
	if s.Art != "" {
		return s.Art
	}
	tile, ok := w.tiles[s.App]
	if !ok {
		tile = placeholder.Art(s.App, placeholder.Media)
		w.tiles[s.App] = tile
	}
	return tile
}

// progress is what surfaces show for a session: the dial seek being
// gathered, else a pending seek's target, else what the source reports.
func (w *worker) progress(s session, now time.Time) snoofer.Progress {
	p := snoofer.Progress{Known: true, Playing: s.playing(), PositionMs: s.PositionMs, DurationMs: s.DurationMs, Rate: s.Rate, At: s.Updated}
	if p.At.IsZero() {
		p.At = now
	}
	if w.scrub != nil && w.scrub.key == s.Key {
		p.PositionMs, p.At = w.scrub.targetMs, w.scrub.last
	} else if pending, ok := w.pending[s.Key]; ok && pending.op == "seek" {
		p.PositionMs, p.At = pending.wantMs, pending.at
	}
	return p
}

func (w *worker) find(key string) (session, bool) {
	return sessionByKey(w.sessions, key)
}

func sessionByKey(sessions []session, key string) (session, bool) {
	for _, s := range sessions {
		if s.Key == key {
			return s, true
		}
	}
	return session{}, false
}

func (w *worker) handle(r snoofer.Request, now time.Time) {
	focused, ok := w.find(w.focus)
	switch r.ID {
	case "nowplaying.deck-media":
		next := w.settings
		next.DeckMediaOff = !next.DeckMediaOff
		w.saveErr = ""
		if err := w.save(next); err != nil {
			w.saveErr = err.Error()
		}
		return
	case "nowplaying.dial":
		if r.Operation == "reset" { // Reset focus: follow playback again.
			w.chosen = false
			w.merge(now)
			return
		}
		if !ok {
			return
		}
		if r.Operation == "adjust" {
			if !focused.CanSeek {
				return
			}
			// Detents gather from the shown position into one seek.
			target := w.progress(focused, now).PositionAt(now) + int64(r.Delta)*seekStep
			if focused.DurationMs > 0 {
				target = min(target, focused.DurationMs-1000)
			}
			w.scrub = &scrub{key: focused.Key, targetMs: max(0, target), last: now}
			return
		}
		w.command(focused, "toggle", 0, now)
		return
	case "nowplaying.toggle", "nowplaying.next", "nowplaying.prev", "nowplaying.mute":
		if ok {
			w.command(focused, strings.TrimPrefix(r.ID, "nowplaying."), 0, now)
		}
		return
	}
	for _, s := range w.sessions {
		if controlID(s.Key) != r.ID {
			continue
		}
		if r.Operation == "hold" { // Held key: choose focus without toggling.
			w.focus, w.chosen = s.Key, true
			return
		}
		if r.Operation == "set" { // The GUI seek slider: an absolute position in ms.
			position, err := strconv.ParseInt(r.Value, 10, 64)
			if err == nil && s.CanSeek && position >= 0 && (s.DurationMs == 0 || position <= s.DurationMs) {
				w.command(s, "seek", position, now)
			}
			return
		}
		w.command(s, "toggle", 0, now)
		return
	}
}

// save persists next. It runs on the worker, never during Start, where the
// host holds the lock saving needs.
func (w *worker) save(next Settings) error {
	raw, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if w.services.SaveSettings == nil {
		return errors.New("settings cannot be saved here")
	}
	if err := w.services.SaveSettings("nowplaying", w.raw, raw); err != nil {
		return err
	}
	w.raw, w.settings = raw, next
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
