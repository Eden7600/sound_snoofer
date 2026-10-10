package hue

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"sound-snoofer/snoofer"
)

// The Decor, Mixed and Saturation dials (openspec/changes/hue-light-groups).
// Turning stages a target and a press applies it, one light at a time,
// because Hue has no group for these lights and limits light commands.

const (
	groupStep      = 5.0  // Brightness percent per tick.
	saturationStep = 0.05 // Saturation fraction per tick.
	xyTolerance    = 0.01 // Confirmation tolerance for colour writes.
)

// groupDial names a light-group dial: its control and which lights it acts on.
type groupDial struct {
	id, label, short, icon string
	lights                 func([]roomLight) []roomLight
	saturation             bool
}

var groupDials = []groupDial{
	{id: "hue.decor", label: "Hue Decorative", short: "Decor", icon: "hue-brightness",
		lights: func(l []roomLight) []roomLight { return withFunction(l, decorative) }},
	{id: "hue.mixed", label: "Hue Mixed", short: "Mixed", icon: "hue-brightness",
		lights: func(l []roomLight) []roomLight { return withFunction(l, mixed) }},
	{id: "hue.saturation", label: "Hue Saturation", short: "Saturation", icon: "hue-saturation", saturation: true,
		lights: colourLights},
}

func findGroupDial(id string) (groupDial, bool) {
	for _, d := range groupDials {
		if d.id == id {
			return d, true
		}
	}
	return groupDial{}, false
}

// stage is a dial's staged target: a brightness percent, or a saturation
// fraction. Nothing is sent until a press, and every stage expires together
// stageHold after the last turn of any light-group dial.
type stage struct {
	target float64
}

// lightWrite is one queued per-light write and how to confirm it.
type lightWrite struct {
	dial, light string
	body        map[string]any
	confirmed   func(roomLight) bool
}

// pendingLight is a written light awaiting the bridge's report.
type pendingLight struct {
	dial      string
	confirmed func(roomLight) bool
	sent      time.Time
}

// lightQueue paces per-light writes: one in flight, at least gap apart.
type lightQueue struct {
	queue    []lightWrite
	inFlight bool
	sending  string // Dial of the write in flight.
	lastSend time.Time
	gap      time.Duration
	waiting  map[string]pendingLight // By light ID.
	errs     map[string]string       // Latest failure per dial.
}

func (w *worker) roomLightsNow() []roomLight {
	if !w.connected || w.settings.Group == "" {
		return nil
	}
	return w.model.roomLights(w.settings.Group)
}

// adjustGroupDial stages on a turn and applies or toggles on a press.
func (w *worker) adjustGroupDial(d groupDial, r snoofer.Request) {
	if !w.connected || w.syncing() {
		return
	}
	lights := d.lights(w.roomLightsNow())
	if len(lights) == 0 {
		return
	}
	if w.stages == nil {
		w.stages = map[string]stage{}
	}
	switch r.Operation {
	case "adjust":
		if r.Delta == 0 {
			return
		}
		base, ok := w.stageBase(d, lights)
		if !ok {
			return
		}
		next := w.stepTarget(d, base, r.Delta)
		w.stages[d.id] = stage{target: next}
		w.stagedAt = time.Now()
	case "press":
		if len(w.stages) > 0 {
			w.applyStages()
			return
		}
		if !d.saturation {
			w.enqueueLights(d.id, toggleWrites(d.id, lights))
		}
	}
}

// applyStages applies every staged target at once and clears the stages.
func (w *worker) applyStages() {
	all := w.roomLightsNow()
	for _, d := range groupDials {
		s, ok := w.stages[d.id]
		if !ok {
			continue
		}
		w.enqueueLights(d.id, w.applyWrites(d, d.lights(all), s.target))
	}
	w.stages = map[string]stage{}
}

// stageBase is where a turn starts: the staged target, or the observed
// group average (brightness 0 when every light is off).
func (w *worker) stageBase(d groupDial, lights []roomLight) (float64, bool) {
	if s, ok := w.stages[d.id]; ok {
		return s.target, true
	}
	if d.saturation {
		return averageSaturation(lights)
	}
	state := summarize(lights)
	switch {
	case !state.Known:
		return 0, false
	case !state.AnyOn:
		return 0, true
	case state.BrightKnown:
		return state.Brightness, true
	}
	return 0, false
}

func (w *worker) stepTarget(d groupDial, base float64, ticks int) float64 {
	if d.saturation {
		return math.Max(0, math.Min(1, base+float64(ticks)*saturationStep))
	}
	return math.Max(minBrightness, math.Min(maxBrightness, base+float64(ticks)*groupStep))
}

// applyWrites builds one write per light for a staged target.
func (w *worker) applyWrites(d groupDial, lights []roomLight, target float64) []lightWrite {
	var out []lightWrite
	for _, l := range lights {
		if !d.saturation {
			out = append(out, lightWrite{dial: d.id, light: l.ID,
				body: map[string]any{"on": map[string]bool{"on": true}, "dimming": map[string]float64{"brightness": target}},
				confirmed: func(r roomLight) bool {
					return r.OnKnown && r.On && r.BrightKnown && math.Abs(r.Brightness-target) <= brightnessStep
				}})
			continue
		}
		if !l.ColourMode {
			continue
		}
		if len(l.Points) > 0 {
			want := make([]xyPoint, len(l.Points))
			points := make([]map[string]any, len(l.Points))
			for n, p := range l.Points {
				want[n] = withSaturation(p, l.Gamut, target)
				points[n] = map[string]any{"color": map[string]any{"xy": map[string]float64{"x": want[n].X, "y": want[n].Y}}}
			}
			out = append(out, lightWrite{dial: d.id, light: l.ID, body: map[string]any{"gradient": map[string]any{"points": points}},
				confirmed: func(r roomLight) bool { return pointsNear(r.Points, want) }})
			continue
		}
		if l.XY == nil {
			continue
		}
		want := withSaturation(*l.XY, l.Gamut, target)
		if want == *l.XY {
			continue // Too near white to have a hue.
		}
		out = append(out, lightWrite{dial: d.id, light: l.ID, body: map[string]any{"color": map[string]any{"xy": map[string]float64{"x": want.X, "y": want.Y}}},
			confirmed: func(r roomLight) bool { return r.XY != nil && near(*r.XY, want) }})
	}
	return out
}

// toggleWrites turns a group off when any light is lit, otherwise on.
func toggleWrites(dial string, lights []roomLight) []lightWrite {
	on := !summarize(lights).AnyOn
	var out []lightWrite
	for _, l := range lights {
		out = append(out, lightWrite{dial: dial, light: l.ID, body: map[string]any{"on": map[string]bool{"on": on}},
			confirmed: func(r roomLight) bool { return r.OnKnown && r.On == on }})
	}
	return out
}

func near(a, b xyPoint) bool { return math.Hypot(a.X-b.X, a.Y-b.Y) <= xyTolerance }

func pointsNear(got, want []xyPoint) bool {
	if len(got) != len(want) {
		return false
	}
	for n := range want {
		if !near(got[n], want[n]) {
			return false
		}
	}
	return true
}

// enqueueLights replaces a dial's queued writes with writes.
func (w *worker) enqueueLights(dial string, writes []lightWrite) {
	q := &w.lights
	kept := q.queue[:0]
	for _, item := range q.queue {
		if item.dial != dial {
			kept = append(kept, item)
		}
	}
	q.queue = append(kept, writes...)
	if q.gap == 0 {
		q.gap = w.timing.lightGap
	}
	if q.errs != nil {
		delete(q.errs, dial)
	}
}

// tickLights sends the next queued write, reports unconfirmed lights and
// drops expired stages. It reports whether anything changed.
func (w *worker) tickLights(ctx context.Context, now time.Time) bool {
	changed := false
	if len(w.stages) > 0 && now.Sub(w.stagedAt) > w.timing.stageHold {
		w.stages = map[string]stage{}
		changed = true
	}
	q := &w.lights
	if w.connected && len(q.queue) > 0 && !q.inFlight && now.Sub(q.lastSend) >= q.gap {
		w.sendLight(ctx, q.queue[0])
		q.queue = q.queue[1:]
		changed = true
	}
	failed := map[string]int{}
	for id, p := range q.waiting {
		if now.Sub(p.sent) > w.timing.confirm {
			failed[p.dial]++
			delete(q.waiting, id)
		}
	}
	for dial, count := range failed {
		if q.errs == nil {
			q.errs = map[string]string{}
		}
		noun := "lights"
		if count == 1 {
			noun = "light"
		}
		q.errs[dial] = fmt.Sprintf("Not applied: %d %s", count, noun)
		changed = true
	}
	return changed
}

func (w *worker) sendLight(ctx context.Context, item lightWrite) {
	client, generation := w.client, w.generation
	w.lights.inFlight = true
	w.lights.sending = item.dial
	w.spawn(ctx, func(ctx context.Context) func(context.Context) {
		err := client.Put(ctx, "light", item.light, item.body)
		return func(context.Context) { w.lightWritten(generation, item, err) }
	})
}

func (w *worker) lightWritten(generation int, item lightWrite, err error) {
	if generation != w.generation {
		return
	}
	q := &w.lights
	q.inFlight = false
	q.sending = ""
	q.lastSend = time.Now()
	switch {
	case errors.Is(err, ErrThrottled):
		q.gap = min(w.timing.maxWriteGap, 2*q.gap)
		q.queue = append([]lightWrite{item}, q.queue...) // Retry it after the longer gap.
		return
	case err != nil:
		if q.errs == nil {
			q.errs = map[string]string{}
		}
		q.errs[item.dial] = err.Error()
		return
	}
	q.gap = w.timing.lightGap
	w.bridgeLink.Activity(time.Now())
	if q.waiting == nil {
		q.waiting = map[string]pendingLight{}
	}
	q.waiting[item.light] = pendingLight{dial: item.dial, confirmed: item.confirmed, sent: time.Now()}
	w.observeLights()
}

// observeLights confirms written lights against the latest bridge state.
func (w *worker) observeLights() {
	for id, p := range w.lights.waiting {
		if r, ok := w.model.resources[id]; ok && p.confirmed(lightView(r)) {
			delete(w.lights.waiting, id)
		}
	}
}

// applying reports whether a dial has writes queued, in flight or awaiting
// confirmation.
func (q *lightQueue) applying(dial string) bool {
	if q.inFlight && q.sending == dial {
		return true
	}
	for _, item := range q.queue {
		if item.dial == dial {
			return true
		}
	}
	for _, p := range q.waiting {
		if p.dial == dial {
			return true
		}
	}
	return false
}

// resetGroupDials drops stages and queued writes, as on disconnect.
func (w *worker) resetGroupDials() {
	w.stages = nil
	w.lights = lightQueue{}
}

// groupDialControls publishes the light-group dials for the selected room.
func (w *worker) groupDialControls() []snoofer.Control {
	all := w.roomLightsNow()
	ready := w.services.Live && w.connected && !w.syncing()
	var out []snoofer.Control
	for _, d := range groupDials {
		lights := d.lights(all)
		c := snoofer.Control{ID: d.id, Label: d.label, ShortLabel: d.short, Group: "Hue", Kind: "numeric", Icon: d.icon,
			Operations: []string{"adjust", "press"}, Available: ready && len(lights) > 0,
			Hidden: w.connected && len(lights) == 0, Status: w.lights.errs[d.id]}
		if s, ok := w.stages[d.id]; ok {
			// Staged: the target with an arrow; nothing is sent until a press.
			target := s.target
			if d.saturation {
				target *= 100
			}
			c.Value, c.Subdued = "→ "+percentLabel(target), true
		} else {
			c.Value = groupValue(d, lights)
			if w.lights.applying(d.id) {
				// Wait, like a pending key, until every light is confirmed or reported.
				c.Status, c.Subdued = "Pending", true
			}
		}
		out = append(out, c)
	}
	return out
}

// groupValue is a dial's observed state.
func groupValue(d groupDial, lights []roomLight) string {
	if len(lights) == 0 {
		return "N/A"
	}
	if d.saturation {
		if s, ok := averageSaturation(lights); ok {
			return percentLabel(s * 100)
		}
		return "White"
	}
	state := summarize(lights)
	switch {
	case !state.Known:
		return "N/A"
	case !state.AnyOn:
		return "Off"
	case state.BrightKnown:
		return percentLabel(state.Brightness)
	}
	return "On"
}
