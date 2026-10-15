package googlemeet

import (
	"fmt"
	"slices"
	"time"

	"sound-snoofer/internal/browserbridge"
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
}
type actionKey struct {
	target  target
	control string
}
type controlContext struct {
	target              target
	selection, revision uint64
}
type controlEpoch struct {
	context controlContext
	epoch   uint64
}

func (t target) value() string {
	if t.instance == "" {
		return ""
	}
	return fmt.Sprintf("%s/%d/%d/%s/%s", t.browser, t.generation, t.tab, t.instance, t.call)
}
func (t target) label() string { return fmt.Sprintf("%s · Tab %d", t.browser, t.tab) }

func controlState(c browserbridge.MeetCall, control string) browserbridge.MeetControl {
	if control == "mic" {
		return c.Mic
	}
	if control == "camera" {
		return c.Camera
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
		w.calls[key] = call{target{update.Browser, update.Generation, state.Tab, state.Instance, state.Call}, state, update.Received}
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
		for _, p := range w.pending {
			p.cancel() // The adapter cannot see a changed Snoofer selection.
		}
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
		if !w.eligible(p.command.target, now) {
			p.cancel()
		}
	}
	for key := range w.notes {
		if !w.eligible(key.target, now) {
			delete(w.notes, key)
		}
	}
}
