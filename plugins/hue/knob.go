package hue

import (
	"fmt"
	"math"
	"time"
)

const (
	brightnessStep = 2.0 // Percent per dial tick; also the confirmation tolerance.
	minBrightness  = 1.0 // Rotation never turns the room off.
	maxBrightness  = 100.0
)

// timing bounds group writes and confirmations. Tests shorten it.
type timing struct {
	writeGap    time.Duration // Minimum interval between group writes.
	maxWriteGap time.Duration // Ceiling for rate-limit backoff.
	confirm     time.Duration // Time allowed for the bridge to report a write.
	pairWindow  time.Duration
	retryBase   time.Duration // First reconnect delay; doubles to retryMax.
	retryMax    time.Duration
	rediscover  time.Duration // Delay after no or ambiguous discovery.
}

var defaultTiming = timing{
	writeGap:    250 * time.Millisecond,
	maxWriteGap: 2 * time.Second,
	confirm:     3 * time.Second,
	pairWindow:  30 * time.Second,
	retryBase:   time.Second,
	retryMax:    30 * time.Second,
	rediscover:  60 * time.Second,
}

func nextBrightness(base float64, ticks int) float64 {
	return math.Min(maxBrightness, math.Max(minBrightness, base+float64(ticks)*brightnessStep))
}

func kelvinToMirek(kelvin float64) int { return int(math.Round(1e6 / kelvin)) }

func clampMirek(mirek, low, high int) int { return min(high, max(low, mirek)) }

// nextMirek moves a kelvin base by whole dial ticks; clockwise is cooler. It
// clamps in kelvin first so a long counter-clockwise turn cannot pass zero.
func nextMirek(baseKelvin float64, ticks, low, high int) int {
	kelvin := baseKelvin + float64(ticks)*kelvinStep
	kelvin = math.Min(mirekToKelvin(low), math.Max(mirekToKelvin(high), kelvin))
	return clampMirek(kelvinToMirek(kelvin), low, high)
}

func percentLabel(value float64) string { return fmt.Sprintf("%.0f%%", math.Round(value)) }

// kelvinLabel rounds to the dial step; mirek conversion makes finer digits noise.
func kelvinLabel(kelvin float64) string {
	return fmt.Sprintf("%dK", int(math.Round(kelvin/kelvinStep)*kelvinStep))
}

// groupRequest holds requested grouped_light targets that the bridge has not
// yet confirmed. Only the latest values are kept, so writes never back up.
type groupRequest struct {
	target     string // grouped_light ID the targets belong to.
	on         *bool
	brightness *float64
	mirek      *int
	dirty      bool // Changed since the last write was sent.
	inFlight   bool
	lastSend   time.Time
	gap        time.Duration
}

func (r *groupRequest) pending() bool { return r.on != nil || r.brightness != nil || r.mirek != nil }

func (r *groupRequest) body() map[string]any {
	body := map[string]any{}
	if r.on != nil {
		body["on"] = map[string]bool{"on": *r.on}
	}
	if r.brightness != nil {
		body["dimming"] = map[string]float64{"brightness": *r.brightness}
	}
	if r.mirek != nil {
		body["color_temperature"] = map[string]int{"mirek": *r.mirek}
	}
	return body
}

// confirm clears targets the bridge now reports within one dial step.
func (r *groupRequest) confirm(view groupView) {
	if r.on != nil && view.OnKnown && view.On == *r.on {
		r.on = nil
	}
	if r.brightness != nil && view.BrightKnown && math.Abs(view.Brightness-*r.brightness) <= brightnessStep {
		r.brightness = nil
	}
	if r.mirek != nil && view.Temperature == temperatureKnown && math.Abs(view.Kelvin-mirekToKelvin(*r.mirek)) <= kelvinStep {
		r.mirek = nil
	}
}
