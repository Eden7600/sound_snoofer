package streamdeck

import (
	"bytes"
	"image"
	"testing"
)

func TestMeterRenderingAndStaticKeyCache(t *testing.T) {
	im := image.NewRGBA(image.Rect(0, 0, 200, 100))
	row := meterTop + meterRows/2
	drawMeter(im, 0, knobPresentation{Meter: true, LevelKnown: true, LevelDB: 0})
	// Flat zones: green below -12 dB, amber to -3 dB, red above.
	if im.RGBAAt(10, row) != meterGreenColor || im.RGBAAt(172, row) != meterAmberColor || im.RGBAAt(195, row) != meterRedColor {
		t.Fatal(im.RGBAAt(10, row), im.RGBAAt(172, row), im.RGBAAt(195, row))
	}
	drawMeter(im, 0, knobPresentation{Meter: true, LevelKnown: true, LevelDB: -60})
	if im.RGBAAt(10, row) == meterGreenColor {
		t.Fatal("silence retained signal")
	}
	peaked := knobPresentation{Meter: true, LevelKnown: true, LevelDB: -60, PeakKnown: true, PeakDB: -30, PositionKnown: true, Position: 0.5, ZeroMark: 0.75, Icon: "speaker-mute"}
	drawMeter(im, 0, peaked)
	if tick := im.RGBAAt((panelWidth-1)/2, meterTop-2); tick != lightValue {
		t.Fatal("peak tick missing", tick)
	}
	drawPosition(im, panelInset, dialWidth, peaked)
	if im.RGBAAt(panelInset+dialWidth/4, trackTop+1) != playbackArea || im.RGBAAt(panelInset+dialWidth*7/8, trackTop+1) != trackColor {
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
