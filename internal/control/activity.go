package control

import (
	"math"
	"slices"
	"sort"
	"time"

	"sound-snoofer/internal/config"
)

// activeAfter is how long signal must last before a silent microphone is
// Active again.
const activeAfter = 300 * time.Millisecond

// micStrips maps microphone options to the hardware strip they feed.
var micStrips = map[string]int{"desk": 0, "lav": 1, "webcam": 2}

// micLatch is one microphone's activity: when its current run of silence or
// signal began, and whether it is latched silent.
type micLatch struct {
	quietSince, loudSince time.Time
	silent                bool
}

// activityLatch turns sampled levels into latched silence. Only the worker
// goroutine uses it.
type activityLatch struct {
	mics map[string]*micLatch
}

// observe records one sample. Levels are linear peaks by option; a missing
// option is unknown, which resets its timers and never counts as silent.
// It reports whether the silent set changed.
func (l *activityLatch) observe(a *config.Activity, levels map[string]float32, now time.Time) bool {
	before := l.silent()
	if l.mics == nil {
		l.mics = map[string]*micLatch{}
	}
	for id := range l.mics {
		if _, known := levels[id]; !known {
			delete(l.mics, id)
		}
	}
	for id, level := range levels {
		m := l.mics[id]
		if m == nil {
			m = &micLatch{}
			l.mics[id] = m
		}
		if dbfs(level) < a.SilenceDB {
			m.loudSince = time.Time{}
			if m.quietSince.IsZero() {
				m.quietSince = now
			}
			if now.Sub(m.quietSince) >= time.Duration(a.SilentAfterS)*time.Second {
				m.silent = true
			}
			continue
		}
		m.quietSince = time.Time{}
		if m.loudSince.IsZero() {
			m.loudSince = now
		}
		if m.silent && now.Sub(m.loudSince) >= activeAfter {
			m.silent = false
		}
	}
	return !slices.Equal(before, l.silent())
}

// silent lists the latched-silent options in a stable order.
func (l *activityLatch) silent() []string {
	out := []string{}
	for id, m := range l.mics {
		if m.silent {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func (l *activityLatch) reset() { l.mics = nil }

func dbfs(peak float32) float64 {
	if peak <= 0 {
		return math.Inf(-1)
	}
	return 20 * math.Log10(float64(peak))
}

// inputLevelReader reads pre-fader microphone strip peaks.
type inputLevelReader interface {
	InputLevels(strips []int) map[int]float32
}
