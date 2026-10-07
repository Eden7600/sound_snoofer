package nowplaying

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"sound-snoofer/snoofer"
)

// viewSession is a session's full detail for the GUI.
type viewSession struct {
	ID, Source, App, Title, Artist, Album, Status, Progress string
	PositionMs, DurationMs, AtMs                            int64 // AtMs: when PositionMs was true (Unix ms).
	Rate                                                    float64
	Focused, CanToggle, CanNext, CanPrev, CanSeek, CanMute  bool
	Muted                                                   bool
	Pending                                                 bool
	Failure                                                 string
}

type browserView struct {
	Name, Version string
	Sessions      int
}

// bridgeView reports the browser bridge for the GUI.
type bridgeView struct {
	Port    int
	Error   string // Why the bridge is not listening.
	Refused string // Why the last connection was refused.
}

type statusView struct {
	Bridge   bridgeView
	Windows  string // Connected, or why not.
	Browsers []browserView
	Sessions []viewSession
}

func (w *worker) publish(commands chan snoofer.Request, now time.Time) {
	var controls []snoofer.Control
	view := statusView{Windows: "Connected", Bridge: bridgeView{Port: w.settings.port(), Error: w.bridgeErr}}
	if w.bridge != nil {
		view.Bridge.Refused = w.bridge.lastRefusal()
	}
	if w.winErr != "" {
		view.Windows = w.winErr
	}
	for n, s := range w.sessions {
		id := controlID(s.Key)
		title := s.Title
		if title == "" {
			title = s.App
		}
		status := w.failure[s.Key]
		if _, ok := w.pending[s.Key]; ok {
			status = "Pending"
		}
		controls = append(controls, snoofer.Control{ID: id, Label: title, ShortLabel: title, Group: "Now playing", Collection: "nowplaying.sessions", CollectionLabel: "Media sessions",
			Order: n + 1, Kind: "command", Icon: "media-play", Artwork: w.artwork(s), Value: s.Status, Status: status, Operations: []string{"press", "hold", "set"}, Hidden: w.settings.DeckMediaOff, Available: s.CanToggle || s.CanSeek})
		_, pending := w.pending[s.Key]
		shown := w.progress(s, now)
		view.Sessions = append(view.Sessions, viewSession{ID: id, Source: s.Source, App: s.App, Title: s.Title, Artist: s.Artist, Album: s.Album, Status: s.Status,
			Progress: shown.Text(now), PositionMs: shown.PositionMs, AtMs: shown.At.UnixMilli(), Rate: shown.Rate, DurationMs: s.DurationMs, Focused: s.Key == w.focus, CanToggle: s.CanToggle, CanNext: s.CanNext,
			CanPrev: s.CanPrev, CanSeek: s.CanSeek, CanMute: s.CanMute, Muted: s.Muted, Pending: pending, Failure: w.failure[s.Key]})
	}
	names := make([]string, 0, len(w.browsers))
	for name := range w.browsers {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		view.Browsers = append(view.Browsers, browserView{Name: name, Version: w.browsers[name].Version, Sessions: len(w.browsers[name].Sessions)})
	}

	focused, ok := w.find(w.focus)

	dial := snoofer.Control{ID: "nowplaying.dial", Label: "Now playing", ShortLabel: "Nothing playing", Group: "Now playing", Kind: "numeric", Icon: "media-play",
		Value: "", Operations: []string{"adjust", "press"}, Available: true}
	transport := func(id, label, short, icon string, can bool, value string) snoofer.Control {
		return snoofer.Control{ID: "nowplaying." + id, Label: label, ShortLabel: short, Group: "Now playing", Kind: "command", Icon: icon, Value: value,
			Operations: []string{"press"}, Hidden: !ok || !can || w.settings.DeckMediaOff, Available: ok && can}
	}
	toggleValue, muteValue, dialStatus := "", "", ""
	if ok {
		title := focused.Title
		if title == "" {
			title = focused.App
		}
		// Value changes only with the play state; progress is telemetry, so
		// turning the dial during playback is never rejected as stale.
		dial.ShortLabel, dial.Value, dial.Artwork = title, focused.Status, w.artwork(focused)
		dial.Progress = w.progress(focused, now)
		toggleValue = focused.Status
		if focused.Muted {
			muteValue = "Muted"
		}
		dialStatus = w.failure[focused.Key]
		// No "Pending" here: the progress already shows the requested
		// position, and a changing status would make further turns stale.
	}
	dial.Status = dialStatus
	dial.Hidden = w.settings.DeckMediaOff
	deckMedia := "On"
	if w.settings.DeckMediaOff {
		deckMedia = "Off"
	}
	// The filter key stays visible so media can be turned back on.
	controls = append(controls, snoofer.Control{ID: "nowplaying.deck-media", Label: "Media on the deck", ShortLabel: "Media", Group: "Now playing", Kind: "toggle", Icon: "media-play",
		Value: deckMedia, Status: w.saveErr, Operations: []string{"press"}, Available: w.services.SaveSettings != nil})
	controls = append(controls, dial,
		transport("prev", "Previous track", "Previous", "media-prev", ok && focused.CanPrev, ""),
		transport("toggle", "Play or pause", "Play", "media-play", ok && focused.CanToggle, toggleValue),
		transport("next", "Next track", "Next", "media-next", ok && focused.CanNext, ""),
		transport("mute", "Mute tab", "Mute", "speaker-mute", ok && focused.CanMute, muteValue))

	data, _ := json.Marshal(view) // Plain fields always marshal.
	summary := fmt.Sprintf("%d sessions", len(w.sessions))
	if !w.services.Live {
		summary = "Preview"
	}
	controls = append(controls, snoofer.Control{ID: "nowplaying.status", Label: "Now playing", Group: "Now playing", Kind: "status", Value: summary, ViewData: data, Available: true})
	_ = w.services.Controls.Publish("nowplaying", controls, func(ctx context.Context, r snoofer.Request) error {
		select {
		case commands <- r:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf("now playing busy")
		}
	})
}
