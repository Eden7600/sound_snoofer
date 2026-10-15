package nowplaying

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"sound-snoofer/internal/mediasessions"
	"sound-snoofer/snoofer"
)

func (r *rig) queue(id, op, value string, delta int) (command, <-chan snoofer.Outcome) {
	r.t.Helper()
	c := r.control(id)
	results := make(chan snoofer.Outcome, 2)
	err := r.controls.DispatchWithResult(r.ctx, snoofer.Request{ID: id, Revision: c.Revision, Operation: op, Value: value, Delta: delta}, func(result snoofer.Outcome) { results <- result })
	if err != nil {
		r.t.Fatal(err)
	}
	select {
	case result := <-results:
		r.t.Fatalf("enqueue settled operation: %+v", result)
	default:
	}
	return <-r.commands, results
}

func result(t *testing.T, results <-chan snoofer.Outcome, want snoofer.OutcomeStatus) snoofer.Outcome {
	t.Helper()
	select {
	case got := <-results:
		if got.Status != want {
			t.Fatalf("result %+v, want %s", got, want)
		}
		return got
	case <-time.After(time.Second):
		t.Fatal("completion lost")
	}
	return snoofer.Outcome{}
}

func twoPlayers(r *rig) {
	a := spotify("playing", 0, r.now)
	b := a
	b.ID, b.Instance, b.Title, b.Status = "Other", "object-other", "Other track", "paused"
	r.win.sessions = []mediasessions.Session{a, b}
	r.tick(0)
}

func TestQueuedFocusDoesNotRetarget(t *testing.T) {
	for _, aba := range []bool{false, true} {
		r := newRig(t, true)
		twoPlayers(r)
		c, results := r.queue("nowplaying.next", "press", "", 0)
		r.w.focus, r.w.chosen = "windows:Other", true
		r.w.merge(r.now)
		if aba {
			r.w.focus = "windows:Spotify.exe"
			r.w.merge(r.now)
		}
		r.w.process(r.ctx, c, r.now)
		result(t, results, snoofer.Rejected)
		if len(r.win.commands) != 0 {
			t.Fatal("queued command retargeted", r.win.commands)
		}
	}
}

func TestQueuedSessionRejectsChangedLifetimeAndTrack(t *testing.T) {
	for _, change := range []string{"instance", "removed", "reopen", "track", "capability", "reorder"} {
		t.Run(change, func(t *testing.T) {
			r := newRig(t, true)
			twoPlayers(r)
			c, results := r.queue(controlID("windows:Spotify.exe"), "press", "", 0)
			switch change {
			case "instance":
				r.win.sessions[0].Instance = "replacement"
			case "removed":
				saved := r.win.sessions
				r.win.sessions = nil
				r.tick(0)
				r.win.sessions = saved
			case "reopen":
				r.w.closeWindows()
			case "track":
				r.win.sessions[0].Title = "New track"
			case "capability":
				r.win.sessions[0].CanPause = false
			case "reorder":
				r.win.sessions[0].Instance, r.win.sessions[1].Instance = r.win.sessions[1].Instance, r.win.sessions[0].Instance
			}
			r.tick(0)
			r.w.process(r.ctx, c, r.now)
			result(t, results, snoofer.Rejected)
			if len(r.win.commands) != 0 {
				t.Fatal("obsolete context wrote", r.win.commands)
			}
		})
	}
}

func TestNativeOutcomesAndExplicitCapturedPlayback(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want snoofer.OutcomeStatus
	}{
		{nil, snoofer.Processed}, {errors.New("native failed"), snoofer.Failed}, {mediasessions.ErrDeclined, snoofer.Failed}, {mediasessions.ErrTimeout, snoofer.Unknown},
	} {
		r := newRig(t, true)
		s := spotify("playing", 0, r.now)
		s.Instance = "native-object"
		r.win.sessions = []mediasessions.Session{s}
		r.tick(0)
		c, results := r.queue("nowplaying.toggle", "press", "", 0)
		r.win.sessions[0].Status = "paused"
		r.tick(0)
		r.win.err = tc.err
		r.w.process(r.ctx, c, r.now)
		result(t, results, tc.want)
		if len(r.win.commands) != 1 || r.win.commands[0] != "native-object:pause:0s" {
			t.Fatal("intent/token changed", r.win.commands)
		}
		if tc.err == nil && len(r.w.pending) != 1 {
			t.Fatal("native call claimed observed completion")
		}
		r.tick(4 * time.Second)
		if len(r.win.commands) != 1 {
			t.Fatal("native operation replayed")
		}
	}
}

func TestMissingNativeIdentityIsUnavailable(t *testing.T) {
	r := newRig(t, true)
	s := spotify("playing", 0, r.now)
	s.Instance = ""
	r.win.sessions = []mediasessions.Session{s}
	r.tick(0)
	if r.control("nowplaying.toggle").Available || r.members()[0].Available {
		t.Fatal("old companion is actionable")
	}
	if !strings.Contains(r.members()[0].Status, "Update") {
		t.Fatal("missing update explanation")
	}
	var view statusView
	if err := json.Unmarshal(r.control("nowplaying.status").ViewData, &view); err != nil {
		t.Fatal(err)
	}
	shown := view.Sessions[0]
	if shown.CanToggle || shown.CanSeek || shown.CanNext || !strings.Contains(shown.Failure, "Update") {
		t.Fatal("GUI view hides companion incompatibility", shown)
	}
	c, results := r.queue("nowplaying.dial", "press", "", 0)
	r.w.process(r.ctx, c, r.now)
	result(t, results, snoofer.Rejected)
	if len(r.win.commands) != 0 {
		t.Fatal(r.win.commands)
	}
}

func TestSeekSubmissionKeepsTargetAndAccumulates(t *testing.T) {
	r := newRig(t, true)
	r.win.sessions = []mediasessions.Session{spotify("paused", 60000, r.now)}
	r.tick(0)
	first, a := r.queue("nowplaying.dial", "adjust", "", 1)
	second, b := r.queue("nowplaying.dial", "adjust", "", 1)
	r.w.process(r.ctx, first, r.now)
	r.w.process(r.ctx, second, r.now)
	result(t, a, snoofer.Submitted)
	result(t, b, snoofer.Submitted)
	if r.w.scrub.targetMs != 70000 || len(r.win.commands) != 0 {
		t.Fatal("detents lost or wrote early")
	}
	r.win.sessions[0].Instance = "replacement"
	r.tick(scrubSettle)
	if len(r.win.commands) != 0 || r.w.scrub != nil {
		t.Fatal("deferred seek retargeted")
	}
}

func TestNowPlayingSettingsOutcomesAndStaleQueue(t *testing.T) {
	r := newRig(t, true)
	r.w.raw = json.RawMessage(`{}`)
	writes := 0
	r.w.services.SaveSettings = func(string, json.RawMessage, json.RawMessage) error { writes++; return nil }
	r.tick(0)
	first, a := r.queue("nowplaying.deck-media", "press", "", 0)
	second, b := r.queue("nowplaying.deck-media", "press", "", 0)
	r.w.process(r.ctx, first, r.now)
	r.w.process(r.ctx, second, r.now)
	result(t, a, snoofer.Processed)
	result(t, b, snoofer.Rejected)
	if writes != 1 || !r.w.settings.DeckMediaOff {
		t.Fatal("settings target changed")
	}
	r.w.services.SaveSettings = func(string, json.RawMessage, json.RawMessage) error { return errors.New("save denied") }
	r.tick(0)
	c, results := r.queue("nowplaying.deck-media", "press", "", 0)
	r.w.process(r.ctx, c, r.now)
	if got := result(t, results, snoofer.Failed); got.Detail != "save denied" {
		t.Fatal(got)
	}
	if !r.w.settings.DeckMediaOff {
		t.Fatal("failed save changed settings")
	}
}

func browserPlayer(r *rig, generation uint64, instance string) {
	r.generations["Brave"] = generation
	r.w.browser(browserUpdate{Browser: "Brave", Generation: generation, Connected: true, Sessions: []browserSession{{ID: "7:0", Instance: instance, Title: "Video", State: "playing", CanNext: true, DurationMs: 100000, CanSeek: true}}})
	r.tick(0)
}

func TestBrowserGenerationAndDocumentChangesRejectCapturedWork(t *testing.T) {
	for _, change := range []string{"connection", "document", "disconnect"} {
		t.Run(change, func(t *testing.T) {
			r := newRig(t, true)
			browserPlayer(r, 1, "document-a")
			c, results := r.queue("nowplaying.next", "press", "", 0)
			switch change {
			case "connection":
				browserPlayer(r, 2, "document-a")
			case "document":
				browserPlayer(r, 1, "document-b")
			case "disconnect":
				r.generations["Brave"] = 0
			}
			r.w.process(r.ctx, c, r.now)
			result(t, results, snoofer.Rejected)
			if len(r.sent) != 0 {
				t.Fatal("browser target replaced", r.sent)
			}
		})
	}
	r := newRig(t, true)
	browserPlayer(r, 2, "current")
	r.w.browser(browserUpdate{Browser: "Brave", Generation: 1, Connected: true, Sessions: []browserSession{{ID: "7:0", Instance: "old", Title: "Old"}}})
	r.w.browser(browserUpdate{Browser: "Brave", Generation: 1})
	r.w.merge(r.now)
	if r.w.browsers["Brave"].Generation != 2 || r.w.sessions[0].Instance != "current" {
		t.Fatal("old update replaced current browser")
	}
}

func TestBrowserBoundariesAndCapturedIntent(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want snoofer.OutcomeStatus
	}{{nil, snoofer.Submitted}, {errBrowserChanged, snoofer.Rejected}, {errors.New("socket lost"), snoofer.Unknown}} {
		r := newRig(t, true)
		browserPlayer(r, 1, "document-a")
		c, results := r.queue("nowplaying.mute", "press", "", 0)
		r.w.send = func(browser string, got browserCommand) error {
			if browser != "Brave" || got.Instance != "document-a" || got.Generation != 1 || got.Value != 1 {
				t.Fatal(got)
			}
			return tc.err
		}
		r.w.process(r.ctx, c, r.now)
		result(t, results, tc.want)
		if tc.err == nil && !r.w.pending[c.target.Key].wantMuted {
			t.Fatal("browser submit claimed observed state")
		}
	}
}

func TestNowPlayingQueueFullStopAndLateCompletion(t *testing.T) {
	r := newRig(t, true)
	twoPlayers(r)
	control := r.control("nowplaying.next")
	request := snoofer.Request{ID: control.ID, Revision: control.Revision, Operation: "press"}
	results := make(chan snoofer.Outcome, 16)
	for n := 0; n < cap(r.commands); n++ {
		if err := r.controls.DispatchWithResult(r.ctx, request, func(v snoofer.Outcome) { results <- v }); err != nil {
			t.Fatal(err)
		}
	}
	full := make(chan snoofer.Outcome, 2)
	if err := r.controls.DispatchWithResult(r.ctx, request, func(v snoofer.Outcome) { full <- v }); err == nil {
		t.Fatal("full queue accepted")
	}
	result(t, full, snoofer.Rejected)
	r.cancel()
	for n := 0; n < cap(r.commands); n++ {
		result(t, results, snoofer.Unknown)
	}
	for len(r.commands) > 0 {
		r.w.process(r.ctx, <-r.commands, r.now)
	}
	if len(r.win.commands) != 0 {
		t.Fatal("cancelled work ran")
	}
	select {
	case got := <-results:
		t.Fatal("duplicate completion", got)
	default:
	}
	select {
	case got := <-full:
		t.Fatal("full queue completed twice", got)
	default:
	}
}

func TestNowPlayingValidationFocusAndPreviewOutcomes(t *testing.T) {
	r := newRig(t, true)
	twoPlayers(r)
	c, results := r.queue(controlID("windows:Other"), "hold", "", 0)
	r.w.process(r.ctx, c, r.now)
	result(t, results, snoofer.Processed)
	if r.w.focus != "windows:Other" || !r.w.chosen {
		t.Fatal("focus not selected")
	}
	c, results = r.queue(controlID("windows:Spotify.exe"), "set", "999999", 0)
	r.w.process(r.ctx, c, r.now)
	result(t, results, snoofer.Rejected)
	r.tick(0)
	c, results = r.queue("nowplaying.next", "press", "", 0)
	r.w.services.Live = false
	r.w.process(context.Background(), c, r.now)
	result(t, results, snoofer.Rejected)
	if len(r.win.commands) != 0 {
		t.Fatal("invalid/preview operation wrote")
	}
}
