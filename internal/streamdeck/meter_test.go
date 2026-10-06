package streamdeck

import (
	"bytes"
	"image"
	"testing"
)

func TestMeterRenderingAndStaticKeyCache(t *testing.T) {
	im := image.NewRGBA(image.Rect(0, 0, 200, 100))
	drawMeter(im, 8, knobPresentation{Meter: true, LevelKnown: true, LevelDB: 0})
	green := im.RGBAAt(8, 70)
	amber := im.RGBAAt(8+20*7, 70)
	red := im.RGBAAt(8+23*7, 70)
	if green.G <= green.R || amber.R <= amber.B || red.R <= red.G {
		t.Fatal(green, amber, red)
	}
	drawMeter(im, 8, knobPresentation{Meter: true, LevelKnown: true, LevelDB: -60})
	if im.RGBAAt(8, 70) == green {
		t.Fatal("silence retained signal")
	}
	peaked := knobPresentation{Meter: true, LevelKnown: true, LevelDB: -60, PeakKnown: true, PeakDB: -30, PositionKnown: true, Position: 0.5, ZeroMark: 0.75}
	drawMeter(im, 8, peaked)
	if tick := im.RGBAAt(8+(dialWidth-1)/2, meterTop+meterRows/2); tick != textColor {
		t.Fatal("peak tick missing", tick)
	}
	drawPosition(im, 8, dialWidth, peaked)
	if im.RGBAAt(8+dialWidth/4, 55) != activeColor || im.RGBAAt(8+dialWidth*7/8, 55) != meterQuietColor {
		t.Fatal("position track wrong")
	}
	f := Frame{}
	f.Dials[0] = Tile{Label: "A1 gain", Value: "0.0 dB", Meter: true, LevelKnown: true, LevelDB: -30}
	keys, touch := renderFrame(f)
	next := f
	next.Dials[0].LevelDB = -1
	cached, changed := renderChangedFrame(next, f, keys)
	if bytes.Equal(touch, changed) {
		t.Fatal("meter did not move")
	}
	for n := range keys {
		if &keys[n][0] != &cached[n][0] {
			t.Fatal("static key was rerendered", n)
		}
	}
	next.Dials[0].LevelKnown = false
	_, unknown := renderChangedFrame(next, f, keys)
	if bytes.Equal(unknown, touch) || bytes.Equal(unknown, changed) {
		t.Fatal("unavailable indistinguishable")
	}
}
