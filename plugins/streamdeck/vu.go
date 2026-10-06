package streamdeck

import (
	"regexp"
	"strconv"
	"time"

	device "sound-snoofer/internal/streamdeck"
)

// VU ballistics for dial meters: instant attack, steady release and a brief
// peak hold. They shape presentation only; readings themselves are untouched.
const (
	vuRelease  = 24.0 // dB per second.
	peakHold   = 1500 * time.Millisecond
	peakFall   = 18.0 // dB per second after the hold.
	meterFloor = -60.0
	gainMin    = -60.0
	gainMax    = 12.0

	meterRefresh = 60 * time.Millisecond  // While a shown dial has a meter or a key animates.
	idleRefresh  = 150 * time.Millisecond // Otherwise.
)

// vuMeter is one dial's displayed level and peak. The Stream Deck plugin
// goroutine owns it.
type vuMeter struct {
	control      string
	known        bool
	level, peak  float64
	at, peakFrom time.Time
}

// apply smooths a tile's fresh reading. A different control or an unknown
// reading resets the meter, so expired readings never decay into fake silence.
func (v *vuMeter) apply(control string, t device.Tile, now time.Time) device.Tile {
	if control != v.control || !t.Meter || !t.LevelKnown {
		*v = vuMeter{control: control}
	}
	if !t.Meter || !t.LevelKnown {
		return t
	}
	reading := max(meterFloor, t.LevelDB)
	if !v.known {
		v.known, v.level, v.peak, v.at, v.peakFrom = true, reading, reading, now, now
	}
	dt := now.Sub(v.at).Seconds()
	v.at = now
	if reading >= v.level {
		v.level = reading
	} else {
		v.level = max(reading, v.level-vuRelease*dt)
	}
	switch {
	case v.level >= v.peak:
		v.peak, v.peakFrom = v.level, now
	case now.Sub(v.peakFrom) > peakHold:
		v.peak = max(v.level, v.peak-peakFall*dt)
	}
	t.LevelDB, t.PeakDB, t.PeakKnown = v.level, v.peak, true
	return t
}

var (
	dbValue      = regexp.MustCompile(`^(-?\d+(?:\.\d+)?) dB$`)
	percentValue = regexp.MustCompile(`(\d+(?:\.\d+)?)%`)
)

// withPosition adds the knob-position track for values with a known range:
// gain in dB (−60…+12, with the zero mark) or a percentage. It reads the same
// displayed value the dial already shows; it never drives behavior.
func withPosition(t device.Tile) device.Tile {
	if m := dbValue.FindStringSubmatch(t.Value); m != nil {
		db, _ := strconv.ParseFloat(m[1], 64)
		t.Position = min(1, max(0, (db-gainMin)/(gainMax-gainMin)))
		t.PositionKnown = true
		t.ZeroMark = (0 - gainMin) / (gainMax - gainMin)
		return t
	}
	if m := percentValue.FindStringSubmatch(t.Value); m != nil {
		pct, _ := strconv.ParseFloat(m[1], 64)
		t.Position = min(1, max(0, pct/100))
		t.PositionKnown = true
	}
	return t
}
