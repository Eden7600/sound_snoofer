package streamdeck

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"testing"
)

func TestGotoKeyPresentation(t *testing.T) {
	if keyAccent("HERE", "deck-page", false, false) != activeColor || keyAccent("", "deck-page", false, false) != neutralColor {
		t.Fatal("go-to key accents")
	}
	examples := []struct{ label, value string }{{"Lights", ""}, {"Soundboard", "Here"}, {"Home", ""}}
	sheet := image.NewRGBA(image.Rect(0, 0, 112*len(examples), 112))
	for n, e := range examples {
		im, err := jpeg.Decode(bytes.NewReader(renderArtwork([]string{e.label, e.value}, 112, 112, false, "deck-page", false, "")))
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
