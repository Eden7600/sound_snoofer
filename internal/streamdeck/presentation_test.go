package streamdeck

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"

	"sound-snoofer/snoofer"
)

func TestAudioOptionLabelsAtNativeSize(t *testing.T) {
	raw, err := os.ReadFile("../../app/web/test/fixtures/audio.live.json")
	if err != nil {
		t.Fatal(err)
	}
	var controls []snoofer.Control
	if err := json.Unmarshal(raw, &controls); err != nil {
		t.Fatal(err)
	}
	byID := map[string]snoofer.Control{}
	for _, c := range controls {
		byID[c.ID] = c
	}
	examples := []struct{ id, value, label, icon string }{
		{"audio.mode", "direct", "Direct", "mode-direct"},
		{"audio.mode", "element", "Element", "mode-element"},
		{"audio.monitor", "off", "Off", "monitor"},
		{"audio.monitor", "pre", "Pre", "monitor"},
		{"audio.monitor", "post", "Post", "monitor"},
		{"audio.record-tap", "pre", "Pre", "tap-pre"},
		{"audio.record-tap", "post", "Post", "tap-post"},
	}
	sheet := image.NewRGBA(image.Rect(0, 0, 112*len(examples), 112))
	for n, example := range examples {
		c := byID[example.id]
		label := c.OptionLabels[example.value]
		if label != example.label {
			t.Fatalf("%s %s label = %q, want %q", example.id, example.value, label, example.label)
		}
		im := decodeKey(t, c.ShortLabel, label, example.icon, false, "")
		for y := 0; y < 112; y++ {
			for x := 0; x < 112; x++ {
				sheet.Set(n*112+x, y, im.At(x, y))
			}
		}
	}
	writePreview(t, os.Getenv("SNOOFER_AUDIO_OPTIONS_PREVIEW"), sheet)
}

func TestBorderlessDistinctSignalIcons(t *testing.T) {
	var previous [][]byte
	for _, icon := range []string{"mode-direct", "mode-element", "tap-pre", "tap-post", "mic-stack"} {
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
		raw := render([]string{"Mic processing", "Direct"}, 112, 112, true, icon, false)
		im, err := jpeg.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		// No perimeter: middle-edge pixels match the key's own fill, sampled
		// clear of text and icon (key pixel (100, 60), rotated).
		fr, fg, fb, _ := im.At(60, 11).RGBA()
		for _, p := range []image.Point{{3, 56}, {108, 56}, {56, 3}, {56, 108}} {
			r, g, b, _ := im.At(p.X, p.Y).RGBA()
			if abs(int(r>>8)-int(fr>>8)) > 12 || abs(int(g>>8)-int(fg>>8)) > 12 || abs(int(b>>8)-int(fb>>8)) > 12 {
				t.Fatal("key perimeter differs from its fill", icon, p)
			}
		}
	}
}
