package streamdeck

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
)

func TestHuePresentation(t *testing.T) {
	if keyAccent("ACTIVE", "hue-scene", false, false) != activeColor {
		t.Fatal("active scene lost active color")
	}
	if keyAccent("READY", "hue-scene", false, false) != neutralColor {
		t.Fatal("ready scene must stay neutral")
	}
	if keyAccent("ON", "huesync-sync", false, false) != activeColor || keyAccent("OFF", "huesync-sync", false, false) != neutralColor {
		t.Fatal("sync state colors")
	}
	if keyAccent("OFF", "hue-motion-off", false, false) != neutralColor {
		t.Fatal("motion Off must stay neutral")
	}
	icons := []string{"hue-scene", "hue-brightness", "hue-pair", "huesync-sync", "huesync-mode", "huesync-intensity", "soundboard-overlap", "soundboard-play", "hue-motion", "hue-motion-off", "deck-page"}
	var previous [][]byte
	for _, icon := range icons {
		canvas := image.NewRGBA(image.Rect(0, 0, 448, 448))
		if !drawIcon(canvas, icon, color.RGBA{255, 255, 255, 255}) {
			t.Fatal("missing icon", icon)
		}
		for _, other := range previous {
			if bytes.Equal(canvas.Pix, other) {
				t.Fatal("indistinguishable icons", icon)
			}
		}
		previous = append(previous, append([]byte(nil), canvas.Pix...))
	}
	// Real generated scene artwork from plugins/hue (see SNOOFER_HUE_ART_SAMPLE).
	sceneArt, err := os.ReadFile("../../docs/design/hue-scene-art.png")
	if err != nil {
		t.Fatal(err)
	}
	art := base64.StdEncoding.EncodeToString(sceneArt)
	examples := []struct{ label, value, icon, artwork string }{
		{"Bright", "Ready", "hue-scene", ""},
		{"Focus", "Active", "hue-scene", ""},
		{"Relax", "Wait", "hue-scene", ""},
		{"Storybook", "Ready", "hue-scene", art},
		{"Storybook", "Active", "hue-scene", art},
		{"Brightness", "62%", "hue-brightness", ""},
		{"Pair", "Press button", "hue-pair", ""},
		{"Sync", "On", "huesync-sync", ""},
		{"Sync", "Off", "huesync-sync", ""},
		{"Mode", "games", "huesync-mode", ""},
		{"Intensity", "extreme", "huesync-intensity", ""},
		{"Overlap", "On", "soundboard-overlap", ""},
		{"Motion", "On", "hue-motion", ""},
		{"Motion", "Off", "hue-motion-off", ""},
		{"Motion", "Mixed", "hue-motion", ""},
	}
	sheet := image.NewRGBA(image.Rect(0, 0, 112*len(examples), 112))
	for n, e := range examples {
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
	writePreview(t, os.Getenv("SNOOFER_HUE_PREVIEW"), sheet)

	frame := Frame{}
	frame.Dials[0] = Tile{Label: "Brightness", Value: "62%", Icon: "hue-brightness"}
	frame.Dials[1] = Tile{Label: "Brightness", Value: "Sync 50%", Icon: "hue-brightness"}
	_, raw := renderFrame(frame)
	rotated, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	bounds := rotated.Bounds()
	strip := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
	for y := 0; y < bounds.Dx(); y++ {
		for x := 0; x < bounds.Dy(); x++ {
			strip.Set(x, y, rotated.At(y, bounds.Dy()-1-x))
		}
	}
	// A control's icon identifier must not appear as text on its dial.
	for y := 70; y < 82; y++ {
		for x := 8; x < 192; x++ {
			if r, _, _, _ := strip.At(x, y).RGBA(); r>>8 > 80 {
				t.Fatalf("dial icon rendered as text at %d,%d", x, y)
			}
		}
	}
	writePreview(t, os.Getenv("SNOOFER_HUE_DIAL_PREVIEW"), strip)
}

func writePreview(t *testing.T, path string, im image.Image) {
	t.Helper()
	if path == "" {
		return
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(file, im)
	closeErr := file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
