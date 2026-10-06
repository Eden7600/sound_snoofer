package streamdeck

import (
	"math"
	"testing"
	"time"

	device "sound-snoofer/internal/streamdeck"
)

func TestVUBallistics(t *testing.T) {
	var v vuMeter
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	tile := func(db float64) device.Tile { return device.Tile{Meter: true, LevelKnown: true, LevelDB: db} }
	out := v.apply("audio.gain-mic", tile(-10), now)
	if out.LevelDB != -10 || out.PeakDB != -10 || !out.PeakKnown {
		t.Fatalf("first reading %+v", out)
	}
	out = v.apply("audio.gain-mic", tile(-40), now.Add(500*time.Millisecond))
	if math.Abs(out.LevelDB-(-22)) > 0.01 || out.PeakDB != -10 {
		t.Fatalf("release or hold wrong: %+v", out)
	}
	out = v.apply("audio.gain-mic", tile(-5), now.Add(600*time.Millisecond))
	if out.LevelDB != -5 || out.PeakDB != -5 {
		t.Fatalf("attack not instant: %+v", out)
	}
	out = v.apply("audio.gain-mic", tile(-50), now.Add(2600*time.Millisecond))
	if out.PeakDB >= -5 || out.PeakDB < out.LevelDB {
		t.Fatalf("peak did not fall after hold: %+v", out)
	}
	if out = v.apply("audio.gain-mic", device.Tile{Meter: true}, now.Add(3*time.Second)); out.LevelKnown || out.PeakKnown {
		t.Fatalf("unknown reading kept a level: %+v", out)
	}
	_ = v.apply("audio.gain-mic", tile(-20), now.Add(4*time.Second))
	if out = v.apply("audio.gain-playback", tile(-30), now.Add(4100*time.Millisecond)); out.LevelDB != -30 || out.PeakDB != -30 {
		t.Fatalf("binding change carried state: %+v", out)
	}
}

func TestDialPosition(t *testing.T) {
	cases := []struct {
		value    string
		position float64
		known    bool
		zeroMark bool
	}{
		{"-6.0 dB", 54.0 / 72, true, true},
		{"12.0 dB", 1, true, true},
		{"-60.0 dB", 0, true, true},
		{"62%", 0.62, true, false},
		{"Sync 62%", 0.62, true, false},
		{"Off", 0, false, false},
		{"N/A", 0, false, false},
	}
	for _, c := range cases {
		got := withPosition(device.Tile{Value: c.value})
		if got.PositionKnown != c.known || math.Abs(got.Position-c.position) > 1e-9 || (got.ZeroMark != 0) != c.zeroMark {
			t.Errorf("%q: %+v", c.value, got)
		}
	}
}
