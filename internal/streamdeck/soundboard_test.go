package streamdeck

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
)

func TestSoundboardPresentation(t *testing.T) {
	if keyAccent("PLAYING", "soundboard-play", false, false) != activeColor {
		t.Fatal("Playing lost active color")
	}
	if keyAccent("READY", "soundboard-play", false, false) != neutralColor {
		t.Fatal("Ready must stay neutral")
	}
	examples := []struct{ label, value, icon, artwork string }{
		{"Fah", "Ready", "soundboard-play", ""},
		{"Fah", "Playing", "soundboard-play", ""},
		{"Stop", "", "soundboard-stop", ""},
		{"Clip", "Ready", "soundboard-play", testArtwork(t)},
		{"Clip", "Playing", "soundboard-play", testArtwork(t)},
	}
	sheet := image.NewRGBA(image.Rect(0, 0, 112*len(examples), 112))
	for n, e := range examples {
		canvas := image.NewRGBA(image.Rect(0, 0, 448, 448))
		if !drawIcon(canvas, e.icon, textColor) {
			t.Fatal("missing icon", e.icon)
		}
		raw := renderArtwork([]string{e.label, e.value}, 112, 112, false, e.icon, false, e.artwork)
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
	if path := os.Getenv("SNOOFER_SOUNDBOARD_PREVIEW"); path != "" {
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(file, sheet)
		closeErr := file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
}
