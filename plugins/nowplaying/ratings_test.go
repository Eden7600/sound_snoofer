package nowplaying

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"sound-snoofer/internal/mediasessions"
	"sound-snoofer/snoofer"
)

func ratedSession(t *testing.T, changes map[string]any) browserSession {
	t.Helper()
	fields := map[string]any{"id": "7:0", "instance": "document-a", "tab": 7, "site": "music.youtube.com", "title": "Same title", "artist": "Same artist", "state": "paused", "durationMs": 90000, "canSeek": true, "canNext": true, "canPrev": true, "track": "video-a:1", "rating": "none", "canRate": true}
	for key, value := range changes {
		fields[key] = value
	}
	data, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	var session browserSession
	if err := json.Unmarshal(data, &session); err != nil {
		t.Fatal(err)
	}
	return session
}

func reportRatings(r *rig, sessions ...browserSession) {
	r.w.browser(browserUpdate{Browser: "Brave", Generation: r.generations["Brave"], Connected: true, Sessions: sessions})
	r.tick(0)
}

func TestRatingWireBounds(t *testing.T) {
	for _, fields := range []map[string]any{
		{"track": strings.Repeat("a", 129)}, {"rating": "favorite"}, {"track": ""}, {"rating": "unknown"}, {"rating": ""},
	} {
		if err := validBrowserSessions([]browserSession{ratedSession(t, fields)}); err == nil {
			t.Errorf("accepted invalid rating: %v", fields)
		}
	}
	for _, fields := range []map[string]any{
		{"track": strings.Repeat("a", 128)}, {"track": "", "rating": "", "canRate": false}, {"track": "", "rating": "unknown", "canRate": false},
	} {
		if err := validBrowserSessions([]browserSession{ratedSession(t, fields)}); err != nil {
			t.Errorf("rejected compatible rating: %v: %v", fields, err)
		}
	}
}

func TestRatingControlsShowObservedState(t *testing.T) {
	for _, tc := range []struct {
		rating, like, dislike string
		canRate, hidden       bool
	}{
		{"none", "Off", "Off", true, false}, {"like", "On", "Off", true, false}, {"dislike", "Off", "On", true, false},
		{"unknown", "N/A", "N/A", false, false}, {"like", "On", "Off", false, false}, {"", "N/A", "N/A", false, true},
	} {
		t.Run(tc.rating+tc.like, func(t *testing.T) {
			r := newRig(t, true)
			reportRatings(r, ratedSession(t, map[string]any{"rating": tc.rating, "canRate": tc.canRate}))
			for _, check := range []struct{ id, label, short, value string }{{"like", "Like Track", "Like", tc.like}, {"dislike", "Dislike Track", "Dislike", tc.dislike}} {
				c := r.control("nowplaying." + check.id)
				if c.Label != check.label || c.ShortLabel != check.short || c.Icon != "media-"+check.id || c.Value != check.value || c.Hidden != tc.hidden || c.Available != tc.canRate {
					t.Errorf("control: %+v", c)
				}
			}
			r.w.settings.DeckMediaOff = true
			r.tick(0)
			if !r.control("nowplaying.like").Hidden || !r.control("nowplaying.dislike").Hidden {
				t.Fatal("ratings bypassed media filter")
			}
		})
	}
}

func TestRatingCapturesExplicitDesiredState(t *testing.T) {
	for _, tc := range []struct{ before, action, want string }{{"none", "like", "like"}, {"like", "like", "none"}, {"none", "dislike", "dislike"}, {"dislike", "dislike", "none"}, {"like", "dislike", "dislike"}, {"dislike", "like", "like"}} {
		t.Run(tc.before+"-"+tc.action, func(t *testing.T) {
			r := newRig(t, true)
			reportRatings(r, ratedSession(t, map[string]any{"rating": tc.before}))
			c, results := r.queue("nowplaying."+tc.action, "press", "", 0)
			// A manual state change must not invert the already captured intent.
			reportRatings(r, ratedSession(t, map[string]any{"rating": tc.want}))
			r.w.process(r.ctx, c, r.now)
			result(t, results, snoofer.Submitted)
			if len(r.sent) != 1 {
				t.Fatal("missing single rating command", r.sent)
			}
			data, err := json.Marshal(r.sent[0])
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]any
			if err := json.Unmarshal(data, &wire); err != nil {
				t.Fatal(err)
			}
			if wire["op"] != "rate" || wire["track"] != "video-a:1" || wire["rating"] != tc.want || wire["instance"] != "document-a" || wire["id"] != "7:0" || wire["value"] != float64(0) {
				t.Fatal(wire)
			}
		})
	}
}

func TestRatingRejectsChangedTarget(t *testing.T) {
	for _, change := range []string{"track", "track ABA", "instance", "connection", "focus", "focus ABA", "capability"} {
		t.Run(change, func(t *testing.T) {
			r := newRig(t, true)
			original := ratedSession(t, nil)
			other := ratedSession(t, map[string]any{"id": "8:0", "instance": "document-b", "track": "video-b:1"})
			reportRatings(r, original, other)
			r.w.focus, r.w.chosen = "Brave:7:0", true
			r.tick(0)
			c, results := r.queue("nowplaying.like", "press", "", 0)
			switch change {
			case "track", "track ABA":
				reportRatings(r, ratedSession(t, map[string]any{"track": "video-b:2"}), other)
				if change == "track ABA" {
					reportRatings(r, original, other)
				}
			case "instance":
				reportRatings(r, ratedSession(t, map[string]any{"instance": "replacement"}), other)
			case "connection":
				r.generations["Brave"]++
				reportRatings(r, original, other)
			case "focus", "focus ABA":
				r.w.focus = "Brave:8:0"
				r.tick(0)
				if change == "focus ABA" {
					r.w.focus = "Brave:7:0"
					r.tick(0)
				}
			case "capability":
				reportRatings(r, ratedSession(t, map[string]any{"canRate": false}), other)
			}
			r.w.process(r.ctx, c, r.now)
			result(t, results, snoofer.Rejected)
			if len(r.sent) != 0 {
				t.Fatal("stale rating sent", r.sent)
			}
		})
	}
}

func TestRatingReservationAndPlaybackAreIndependent(t *testing.T) {
	r := newRig(t, true)
	reportRatings(r, ratedSession(t, nil))
	first, a := r.queue("nowplaying.like", "press", "", 0)
	second, b := r.queue("nowplaying.dislike", "press", "", 0)
	r.w.process(r.ctx, first, r.now)
	result(t, a, snoofer.Submitted)
	r.w.process(r.ctx, second, r.now)
	result(t, b, snoofer.Rejected)
	r.tick(0)
	for _, id := range []string{"like", "dislike"} {
		c := r.control("nowplaying." + id)
		if c.Value != "Off" || c.Status != "Pending" || !c.Available {
			t.Fatal("optimistic or unavailable pending rating", c)
		}
	}
	play, played := r.queue("nowplaying.toggle", "press", "", 0)
	r.w.process(r.ctx, play, r.now)
	result(t, played, snoofer.Submitted)
	reportRatings(r, ratedSession(t, map[string]any{"rating": "like"}))
	if c := r.control("nowplaying.like"); c.Value != "On" || c.Status != "" {
		t.Fatal("readback did not confirm", c)
	}
	if len(r.w.pending) != 1 {
		t.Fatal("rating consumed playback confirmation")
	}
	reportRatings(r, ratedSession(t, map[string]any{"rating": "like", "state": "playing"}))
	if len(r.w.pending) != 0 || len(r.sent) != 2 {
		t.Fatal("independent operations did not settle", r.sent)
	}
}

func TestRatingReservationSurvivesFocusAndUnknownState(t *testing.T) {
	r := newRig(t, true)
	original := ratedSession(t, nil)
	other := ratedSession(t, map[string]any{"id": "8:0", "rating": "", "canRate": false})
	reportRatings(r, original, other)
	r.w.focus, r.w.chosen = "Brave:7:0", true
	r.tick(0)
	first, a := r.queue("nowplaying.like", "press", "", 0)
	r.w.process(r.ctx, first, r.now)
	result(t, a, snoofer.Submitted)
	r.w.focus = "Brave:8:0"
	reportRatings(r, ratedSession(t, map[string]any{"rating": "unknown", "canRate": false}), other)
	if !r.control("nowplaying.like").Hidden {
		t.Fatal("unsupported focus redirected ratings")
	}
	r.w.focus = "Brave:7:0"
	reportRatings(r, original, other)
	second, b := r.queue("nowplaying.dislike", "press", "", 0)
	r.w.process(r.ctx, second, r.now)
	result(t, b, snoofer.Rejected)
	if len(r.sent) != 1 {
		t.Fatal("focus/capability freed pending rating", r.sent)
	}
}

func TestRatingTimeoutLateStateAndTrackRetirement(t *testing.T) {
	r := newRig(t, true)
	reportRatings(r, ratedSession(t, nil))
	c, done := r.queue("nowplaying.like", "press", "", 0)
	r.w.process(r.ctx, c, r.now)
	result(t, done, snoofer.Submitted)
	r.tick(3*time.Second + time.Millisecond)
	if c := r.control("nowplaying.like"); c.Value != "Off" || c.Status == "" || c.Status == "Pending" {
		t.Fatal("missing bounded no-confirmation feedback", c)
	}
	reportRatings(r, ratedSession(t, map[string]any{"rating": "like"}))
	if r.control("nowplaying.like").Value != "On" || len(r.sent) != 1 {
		t.Fatal("late actual rating lost or retried")
	}
	c, done = r.queue("nowplaying.dislike", "press", "", 0)
	r.w.process(r.ctx, c, r.now)
	result(t, done, snoofer.Submitted)
	reportRatings(r, ratedSession(t, map[string]any{"track": "video-b:2"}))
	if c := r.control("nowplaying.like"); c.Status != "" || c.Value != "Off" {
		t.Fatal("old feedback crossed tracks", c)
	}
	r.w.browser(browserUpdate{Browser: "Brave", Generation: 1})
	r.tick(0)
	r.generations["Brave"]++
	reportRatings(r, ratedSession(t, nil))
	if r.control("nowplaying.like").Status != "" || len(r.sent) != 2 {
		t.Fatal("rating replayed or retained after reconnect")
	}
}

func TestRatingUncertainSendKeepsReservation(t *testing.T) {
	for _, sendErr := range []error{errors.New("socket write failed"), errBrowserChanged, errBrowserBusy} {
		definite := errors.Is(sendErr, errBrowserChanged) || errors.Is(sendErr, errBrowserBusy)
		r := newRig(t, true)
		reportRatings(r, ratedSession(t, nil))
		r.w.send = func(_ string, c browserCommand) error {
			r.sent = append(r.sent, c)
			return sendErr
		}
		c, done := r.queue("nowplaying.like", "press", "", 0)
		r.w.process(r.ctx, c, r.now)
		want := snoofer.Unknown
		if definite {
			want = snoofer.Rejected
		}
		result(t, done, want)
		c, done = r.queue("nowplaying.dislike", "press", "", 0)
		r.w.process(r.ctx, c, r.now)
		result(t, done, snoofer.Rejected)
		count := 1
		if definite {
			count = 2
		}
		if len(r.sent) != count {
			t.Fatal("uncertain send reservation", definite, r.sent)
		}
	}
}

func TestRatingReloadRequiredDisablesPageTransport(t *testing.T) {
	r := newRig(t, true)
	reportRatings(r, ratedSession(t, map[string]any{"reloadRequired": true, "canRate": false, "rating": "unknown"}))
	for _, id := range []string{controlID("Brave:7:0"), "nowplaying.toggle", "nowplaying.next", "nowplaying.prev", "nowplaying.like", "nowplaying.dislike"} {
		if c := r.control(id); c.Available {
			t.Fatal("reload required control available", c)
		}
	}
	if !r.control("nowplaying.mute").Available {
		t.Fatal("reload disabled independent tab mute")
	}
	var view statusView
	if err := json.Unmarshal(r.control("nowplaying.status").ViewData, &view); err != nil {
		t.Fatal(err)
	}
	if len(view.Sessions) != 1 || view.Sessions[0].Failure != "Reload this tab" || view.Sessions[0].CanSeek {
		t.Fatal(view)
	}
	reportRatings(r, ratedSession(t, map[string]any{"track": "", "rating": "", "canRate": false}))
	if !r.control("nowplaying.toggle").Available || !r.control("nowplaying.like").Hidden {
		t.Fatal("old extension compatibility changed")
	}
}

func TestRatingFollowsFocusedSessionOnly(t *testing.T) {
	r := newRig(t, true)
	r.win.sessions = []mediasessions.Session{spotify("paused", 0, r.now)}
	first := ratedSession(t, nil)
	second := ratedSession(t, map[string]any{"id": "8:0", "instance": "document-b", "track": "video-b:1", "rating": "dislike"})
	reportRatings(r, first, second)
	r.w.focus, r.w.chosen = "Brave:8:0", true
	r.tick(0)
	if r.control("nowplaying.dislike").Value != "On" {
		t.Fatal("rating did not follow second session")
	}
	c, done := r.queue("nowplaying.like", "press", "", 0)
	r.w.process(r.ctx, c, r.now)
	result(t, done, snoofer.Submitted)
	if len(r.sent) != 1 || r.sent[0].ID != "8:0" || r.sent[0].Track != "video-b:1" || r.sent[0].Instance != "document-b" {
		t.Fatal("rating redirected from focused session", r.sent)
	}
	r.w.focus = "windows:Spotify.exe"
	r.tick(0)
	for _, id := range []string{"nowplaying.like", "nowplaying.dislike"} {
		if control := r.control(id); !control.Hidden || control.Available || control.Value != "N/A" {
			t.Fatal("native focus exposed browser rating", control)
		}
		if err := r.controls.Dispatch(r.ctx, snoofer.Request{ID: id, Operation: "press", Revision: r.control(id).Revision}); err == nil {
			t.Fatal("unsupported native rating accepted")
		}
	}
	if len(r.sent) != 1 || len(r.win.commands) != 0 {
		t.Fatal("unsupported focus redirected request", r.sent, r.win.commands)
	}
}

func TestRatingReloadRejectsCapturedSessionActions(t *testing.T) {
	for _, op := range []string{"press", "set"} {
		t.Run(op, func(t *testing.T) {
			r := newRig(t, true)
			reportRatings(r, ratedSession(t, nil))
			c, done := r.queue(controlID("Brave:7:0"), op, "30000", 0)
			reportRatings(r, ratedSession(t, map[string]any{"reloadRequired": true}))
			r.w.process(r.ctx, c, r.now)
			result(t, done, snoofer.Rejected)
			if len(r.sent) != 0 {
				t.Fatal("old captured session action bypassed reload boundary", r.sent)
			}
		})
	}
}

func TestRatingCapabilityRecoveryPreservesPlaybackAndCapturedIntent(t *testing.T) {
	r := newRig(t, true)
	reportRatings(r, ratedSession(t, nil))
	play, played := r.queue("nowplaying.toggle", "press", "", 0)
	r.w.process(r.ctx, play, r.now)
	result(t, played, snoofer.Submitted)
	rating, rated := r.queue("nowplaying.like", "press", "", 0)
	reportRatings(r, ratedSession(t, map[string]any{"rating": "unknown", "canRate": false}))
	if len(r.w.pending) != 1 {
		t.Fatal("rating capability change retired playback confirmation")
	}
	reportRatings(r, ratedSession(t, map[string]any{"rating": "like"}))
	r.w.process(r.ctx, rating, r.now)
	result(t, rated, snoofer.Submitted)
	if len(r.sent) != 2 || r.sent[1].Op != "rate" || r.sent[1].Rating != "like" || len(r.w.pending) != 1 {
		t.Fatal("capability recovery lost captured intent or playback", r.sent, r.w.pending)
	}
	reportRatings(r, ratedSession(t, map[string]any{"rating": "like", "state": "playing"}))
	if len(r.w.pending) != 0 || r.control("nowplaying.like").Status != "" {
		t.Fatal("independent readback did not settle after recovery")
	}
}
