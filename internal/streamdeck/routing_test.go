package streamdeck

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"testing"
)

// Routing keys: a destination key, then source switches. Switches show
// On/Off as state words; the destination shows its state word.
func TestRoutingPresentation(t *testing.T) {
	examples := []struct{ label, value, icon string }{
		{"Music", "In use", "vr-playback"},
		{"Computer", "On", "record-computer"},
		{"Monitor", "Off", "monitor"},
		{"Soundboard", "Off", "soundboard-play"},
		{"Tape", "On", "tape-play"},
		{"Monitor out", "Missing", "vr-playback"},
		{"Phones", "No output", "vr-playback"},
	}
	if keyAccent("ON", "record-computer", false, false) != activeColor || keyAccent("OFF", "monitor", false, false) != neutralColor {
		t.Fatal("switch accents")
	}
	// Destinations match the GUI: In use is active, a missing device is never
	// danger, and No output needs attention.
	if keyAccent("IN USE", "vr-playback", false, false) != activeColor || keyAccent("MISSING", "vr-playback", false, false) != neutralColor || keyAccent("NO OUTPUT", "vr-playback", false, false) != attentionColor {
		t.Fatal("destination accents")
	}
	sheet := image.NewRGBA(image.Rect(0, 0, 112*len(examples), 112))
	for n, e := range examples {
		raw := renderArtwork([]string{e.label, e.value}, 112, 112, false, e.icon, false, "")
		im, err := jpeg.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		// Undo the hardware rotation for the review sheet.
		for y := 0; y < 112; y++ {
			for x := 0; x < 112; x++ {
				sheet.Set(n*112+x, y, im.At(y, 111-x))
			}
		}
	}
	writePreview(t, os.Getenv("SNOOFER_ROUTING_PREVIEW"), sheet)
}
