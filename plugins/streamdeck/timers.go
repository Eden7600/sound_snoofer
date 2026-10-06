package streamdeck

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

// timerPanels places the timers of the providers shown on a page in the
// panels of its unbound dials, oldest first: one per panel while they fit,
// then two, and the newest two per panel beyond that.
func timerPanels(p Page, shown map[string]snoofer.Control, now time.Time) map[int][2]device.TimerTile {
	providers := map[string]bool{}
	for _, b := range append(p.Keys[:], p.Dials[:]...) {
		if provider, _, ok := strings.Cut(b.Control, "."); ok && provider != "streamdeck" {
			providers[provider] = true
		}
	}
	var owners []snoofer.Control
	for _, c := range shown {
		provider, _, _ := strings.Cut(c.ID, ".")
		if len(c.Timers) > 0 && providers[provider] {
			owners = append(owners, c)
		}
	}
	slices.SortFunc(owners, func(a, b snoofer.Control) int { return strings.Compare(a.ID, b.ID) })
	var timers []snoofer.Timer
	for _, c := range owners {
		timers = append(timers, c.Timers...)
	}
	var free []int
	for n, b := range p.Dials {
		if b.Control == "" {
			free = append(free, n)
		}
	}
	if len(timers) == 0 || len(free) == 0 {
		return nil
	}
	perPanel := 1
	if len(timers) > len(free) {
		perPanel = 2
	}
	if limit := perPanel * len(free); len(timers) > limit {
		timers = timers[len(timers)-limit:]
	}
	out := map[int][2]device.TimerTile{}
	for i, t := range timers {
		c := shown[t.Control]
		panel := out[free[i/perPanel]]
		panel[i%perPanel] = device.TimerTile{Artwork: artworkAt(c, now), Icon: c.Icon, Label: c.Label, Time: remaining(t.Ends.Sub(now))}
		out[free[i/perPanel]] = panel
	}
	return out
}

// remaining formats a countdown as m:ss, rounded up and never negative.
func remaining(d time.Duration) string {
	seconds := max(0, int(math.Ceil(d.Seconds())))
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}
