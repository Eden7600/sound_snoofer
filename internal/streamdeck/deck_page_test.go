package streamdeck

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"testing"
)

// TestGotoKeyPresentation covers go-to and scroll keys.
func TestGotoKeyPresentation(t *testing.T) {
	if keyAccent("HERE", "deck-page", false, false) != activeColor || keyAccent("", "deck-page", false, false) != neutralColor {
		t.Fatal("go-to key accents")
	}
	examples := []struct{ label, value, icon string }{{"Lights", "", "deck-page"}, {"Soundboard", "Here", "deck-page"}, {"Up", "1/2", "deck-up"}, {"Down", "1/2", "deck-down"}, {"Game", "40%", "app-audio"}, {"Auto", "", "focus-reset"}}
	sheet := image.NewRGBA(image.Rect(0, 0, 112*len(examples), 112))
	for n, e := range examples {
		im, err := jpeg.Decode(bytes.NewReader(renderArtwork([]string{e.label, e.value}, 112, 112, false, e.icon, false, "")))
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
	writePreview(t, os.Getenv("SNOOFER_DECK_PAGE_PREVIEW"), sheet)
}
