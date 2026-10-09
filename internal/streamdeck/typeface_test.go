package streamdeck

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestEmbeddedFontsParse(t *testing.T) {
	set := typefaces()
	if set.fonts[labelFace] == nil || set.fonts[valueFace] == nil {
		t.Fatal("embedded fonts did not parse")
	}
	if set.width(valueFace, 30, "STRONG") >= set.width(labelFace, 30, "STRONG") {
		t.Fatal("value face is not condensed")
	}
}

func TestLabelsFitWithoutCutting(t *testing.T) {
	set := typefaces()
	// The longest short labels the built-in providers publish, and some names.
	for _, label := range []string{"Echo strength", "Mic processing", "Stop recording", "Google Chrome", "Concentrate", "Soundboard",
		"Record PC", "Record mic", "Reset focus", "Intensity", "Screen share", "Brightness", "Lofi beats", "Playback"} {
		got := set.fit(labelFace, label, keyTextWidth, 16, 12, 2)
		if got.cut {
			t.Errorf("%q was cut: %v at %d px", label, got.lines, got.px)
		}
		for _, line := range got.lines {
			if set.width(labelFace, got.px, line) > keyTextWidth {
				t.Errorf("%q line %q overflows at %d px", label, line, got.px)
			}
		}
	}
	if got := set.fit(labelFace, "Echo strength", keyTextWidth, 16, 12, 2); got.px != 16 || len(got.lines) != 2 {
		t.Fatal("Echo strength should wrap at full size", got)
	}
	if got := set.fit(labelFace, "Mic", keyTextWidth, 16, 12, 2); got.px != 16 || len(got.lines) != 1 {
		t.Fatal("short label changed", got)
	}
}

func TestFitShrinksThenCuts(t *testing.T) {
	set := typefaces()
	if got := set.fit(valueFace, "STRONG", keyTextWidth, 30, 18, 1); got.px < 24 || got.cut {
		t.Fatal("STRONG should fit near full size", got)
	}
	if got := set.fit(valueFace, "BALANCED", keyTextWidth, 30, 18, 1); got.px >= 30 || got.px < 18 || got.cut {
		t.Fatal("value should shrink to fit", got)
	}
	got := set.fit(labelFace, "Supercalifragilisticexpialidocious microphone channel", keyTextWidth, 16, 12, 2)
	if !got.cut || len(got.lines) != 2 || !strings.HasSuffix(got.lines[1], "…") {
		t.Fatal("overlong label not cut on its last line", got)
	}
	for _, line := range got.lines {
		if set.width(labelFace, got.px, line) > keyTextWidth {
			t.Fatal("cut line overflows", line)
		}
	}
}

func TestBitmapFallback(t *testing.T) {
	set := &typeset{}
	got := set.fit(labelFace, "Echo strength", keyTextWidth, 16, 12, 2)
	if len(got.lines) == 0 {
		t.Fatal("fallback produced no lines")
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 448, 448))
	set.draw(canvas, 4, labelFace, got.px, 10, 8, got.lines[0], color.RGBA{255, 255, 255, 255})
	for _, b := range canvas.Pix {
		if b != 0 {
			return
		}
	}
	t.Fatal("fallback drew nothing")
}
