package googlemeet

import (
	"context"
	"fmt"
	"slices"
	"time"

	"sound-snoofer/internal/browserbridge"
	"sound-snoofer/snoofer"
)

type tabKey struct {
	browser string
	tab     int64
}
type target struct {
	browser        string
	generation     uint64
	tab            int64
	instance, call string
}
type call struct {
	target  target
	state   browserbridge.MeetCall
	sampled time.Time
	newCall bool
}
type actionKey struct {
	target  target
	control string
}

func (t target) value() string {
	if t.instance == "" {
		return ""
	}
	return fmt.Sprintf("%s/%d/%d/%s/%s", t.browser, t.generation, t.tab, t.instance, t.call)
}
func (t target) label() string { return fmt.Sprintf("%s · Tab %d", t.browser, t.tab) }
func (t target) peer() string  { return fmt.Sprintf("%s/%d", t.browser, t.tab) }

func controlState(c browserbridge.MeetCall, control string) browserbridge.MeetControl {
	if control == "mic" {
		return c.Mic
	}
	if control == "camera" {
		return c.Camera
	}
	if control == "share" && c.Share != nil {
		return *c.Share
	}
	if control == "leave" && c.Leave != nil {
		return *c.Leave
	}
	return browserbridge.MeetControl{}
}

func (w *worker) receive(update browserbridge.MeetUpdate) {
	if !update.Connected {
		if w.browsers[update.Browser].Generation == update.Generation {
			delete(w.browsers, update.Browser)
		}
		return
	}
	if update.Generation == 0 || w.connection.Generation(update.Browser) != update.Generation {
		return
	}
	previous := w.browsers[update.Browser]
	if previous.Generation == update.Generation && update.Calls == nil {
		update.Calls = previous.Calls // Hello metadata is not an observed empty list.
	}
	w.browsers[update.Browser] = update
	present := map[tabKey]bool{}
	for _, state := range update.Calls {
		key := tabKey{update.Browser, state.Tab}
		present[key] = true
		old, exists := w.calls[key]
		if exists && previous.Generation == update.Generation && old.state.Instance == state.Instance {
			// A repeated cached sample cannot extend its original freshness.
			if state.Sample <= old.state.Sample {
				continue
			}
		}
		t := target{update.Browser, update.Generation, state.Tab, state.Instance, state.Call}
		// Missing/stale reports and new documents do not prove that a call ended.
		newCall := state.State == "joined" && exists && old.target.generation == update.Generation && old.state.Instance == state.Instance &&
			update.Received.Sub(old.sampled) < freshness && (old.state.State == "prejoin" || old.state.State == "joined" && old.state.Call != state.Call)
		w.calls[key] = call{target: t, state: state, sampled: update.Received, newCall: newCall || old.target == t && old.newCall}
	}
	for key := range w.calls {
		if key.browser == update.Browser && !present[key] {
			delete(w.calls, key)
		}
	}
	w.link.Activity(update.Received)
}

func (w *worker) current(t target, now time.Time) (call, bool) {
	c, ok := w.calls[tabKey{t.browser, t.tab}]
	browser := w.browsers[t.browser]
	return c, ok && c.target == t && browser.Supported && browser.Generation == t.generation &&
		w.connection.Generation(t.browser) == t.generation && c.state.State == "joined" && t.call != "" &&
		!c.sampled.IsZero() && now.Sub(c.sampled) < freshness
}

func (w *worker) eligible(t target, now time.Time) bool { _, ok := w.current(t, now); return ok }

func (w *worker) selectTarget(t target) {
	if w.selected != t {
		w.selected = t
		w.selection++
	}
}

func (w *worker) reconcile(now time.Time) {
	for browser, metadata := range w.browsers {
		if w.connection.Generation(browser) != metadata.Generation {
			delete(w.browsers, browser)
		}
	}
	for key := range w.calls {
		if _, ok := w.browsers[key.browser]; !ok {
			delete(w.calls, key)
		}
	}
	var choices []target
	for _, c := range w.calls {
		if w.eligible(c.target, now) {
			choices = append(choices, c.target)
		}
	}
	slices.SortFunc(choices, func(a, b target) int {
		if a.browser != b.browser {
			if a.browser < b.browser {
				return -1
			}
			return 1
		}
		if a.tab < b.tab {
			return -1
		}
		if a.tab > b.tab {
			return 1
		}
		return 0
	})
	if !slices.Equal(choices, w.choices) {
		w.choices = choices
		w.catalogue++
	}
	if w.selected != (target{}) && !w.eligible(w.selected, now) {
		w.selectTarget(target{})
	}
	if !w.discovered && len(choices) > 0 {
		w.discovered = true
		if len(choices) == 1 {
			w.selectTarget(choices[0])
		}
	}
	for _, p := range w.pending {
		// Leave confirmation can follow the snapshot that removes its call.
		if p.control != "leave" && !w.eligible(p.target, now) {
			p.cancel()
		}
	}
	for key := range w.notes {
		if !w.eligible(key.target, now) {
			delete(w.notes, key)
		}
	}
}

func meetingValue(state browserbridge.MeetControl) snoofer.MeetingValue {
	v := snoofer.MeetingValue{Identity: state.Identity, Known: state.Value != nil, Available: state.Available && state.Identity != 0, Revision: state.Revision}
	if state.Value != nil {
		v.Value = *state.Value
	}
	return v
}

func (w *worker) synchronize(ctx context.Context, now time.Time) {
	peers := make([]snoofer.MeetingPeer, 0, len(w.choices))
	for _, t := range w.choices {
		c, ok := w.current(t, now)
		if !ok {
			continue
		}
		peers = append(peers, snoofer.MeetingPeer{ID: t.peer(), Lifetime: t.value(), Label: "Meet · " + t.label(), Joined: true, NewCall: c.newCall,
			Mute: meetingValue(c.state.Mic), Camera: meetingValue(c.state.Camera),
			Share: meetingValue(controlState(c.state, "share")), Leave: meetingValue(controlState(c.state, "leave"))})
	}
	// Reports wake all adapters; unchanged snapshots must not feed that wake back.
	if !slices.Equal(peers, w.lastReport) {
		if err := w.meetings.Report(peers); err != nil {
			w.link.Fail(err.Error(), now)
			return
		}
		w.lastReport = peers
	}
	for key, c := range w.calls {
		c.newCall = false
		w.calls[key] = c
	}
	shared := w.services.Meetings.Snapshot()
	for key := range w.notes {
		current, ok := w.current(key.target, now)
		state := controlState(current.state, key.control)
		target := shared.Mute
		if key.control == "camera" {
			target = shared.Camera
		}
		if !ok || !state.Available || state.Identity == 0 || state.Value == nil || !target.Known || *state.Value != target.Value {
			continue
		}
		pending := false
		for _, p := range w.pending {
			pending = pending || p.target == key.target && p.control == key.control
		}
		if !pending {
			delete(w.notes, key)
		}
	}
	if !w.services.Live || ctx.Err() != nil {
		return
	}
	for capacity := 8 - len(w.pending); capacity > 0; capacity-- {
		if action, ok := w.meetings.NextAction(); ok {
			w.applyAction(ctx, action, now)
			continue
		}
		c, ok := w.meetings.Next()
		if !ok {
			return
		}
		w.apply(ctx, c, now)
	}
}
