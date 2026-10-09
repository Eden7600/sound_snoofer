package streamdeck

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"
)

// decodeKey renders a key and undoes the hardware rotation.
func decodeKey(t *testing.T, label, value, icon string, on bool, artwork string) *image.RGBA {
	t.Helper()
	raw := renderArtwork([]string{label, value}, 112, 112, on, icon, false, artwork)
	im, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	out := image.NewRGBA(image.Rect(0, 0, 112, 112))
	for y := 0; y < 112; y++ {
		for x := 0; x < 112; x++ {
			out.Set(x, y, im.At(y, 111-x))
		}
	}
	return out
}

func near(a, b color.RGBA, tolerance int) bool {
	d := func(x, y uint8) int { return abs(int(x) - int(y)) }
	return d(a.R, b.R) <= tolerance && d(a.G, b.G) <= tolerance && d(a.B, b.B) <= tolerance
}

func TestKeysKeepCornersClear(t *testing.T) {
	// Dark keys with the longest content: nothing but background within 8 px
	// of any edge, where the rounded keycap hides pixels.
	for _, k := range []struct{ label, value, icon string }{
		{"Echo strength", "Balanced", "echo"},
		{"Mic processing", "Pre", "tap-pre"},
		{"Google Chrome", "100%", "app-audio"},
		{"Soundboard", "", "deck-page"},
		{"Record PC", "Off", "record-computer"},
	} {
		im := decodeKey(t, k.label, k.value, k.icon, false, "")
		for y := 0; y < 112; y++ {
			for x := 0; x < 112; x++ {
				if x >= 8 && x < 104 && y >= 8 && y < 104 {
					continue
				}
				if !near(im.RGBAAt(x, y), keyColor, 24) {
					t.Fatalf("%s: pixel (%d, %d) is %v inside the corner inset", k.label, x, y, im.RGBAAt(x, y))
				}
			}
		}
	}
}

func TestAreaColoursAndStates(t *testing.T) {
	// Fills sample the key's middle-right, clear of text and icon.
	fillAt := func(im *image.RGBA) color.RGBA { return im.RGBAAt(100, 60) }
	if got := fillAt(decodeKey(t, "Playback", "On", "speaker-mute", true, "")); !near(got, playbackArea, 12) {
		t.Fatal("Playback on is not filled blue", got)
	}
	if got := fillAt(decodeKey(t, "Mic", "Muted", "mic-mute-muted", true, "")); !near(got, criticalColor, 12) {
		t.Fatal("muted mic is not filled red", got)
	}
	off := decodeKey(t, "Record PC", "Off", "record-computer", false, "")
	if got := fillAt(off); !near(got, keyColor, 12) {
		t.Fatal("off key is not dark", got)
	}
	// The label's area colour: some pixel in the label row is near violet.
	violet := false
	for x := 10; x < 80 && !violet; x++ {
		for y := 8; y < 26 && !violet; y++ {
			violet = near(off.RGBAAt(x, y), recorderArea, 40)
		}
	}
	if !violet {
		t.Fatal("off recorder label is not violet")
	}
	failed := decodeKey(t, "Processing", "Error", "mode-element", false, "")
	if got := failed.RGBAAt(56, 2); !near(got, criticalColor, 30) {
		t.Fatal("error frame missing", got)
	}
	if got := failed.RGBAAt(1, 1); near(got, criticalColor, 60) {
		t.Fatal("error frame is square in the corner", got)
	}
	if got := failed.RGBAAt(12, 46); !near(got, criticalColor, 30) {
		t.Fatal("alert square missing", got)
	}
	pending := decodeKey(t, "Sync", "Wait", "huesync-sync", false, "")
	if got := pending.RGBAAt(4, 2); !near(got, attentionColor, 30) {
		t.Fatal("pending dashes missing", got)
	}
	if keyArea("hue-scene") != lightsArea || keyArea("soundboard-play") != soundboardArea || keyArea("app-audio") != mediaArea ||
		keyArea("mic-mute-muted") != micArea || keyArea("tape-ff") != recorderArea || keyArea("deck-page") != otherArea || keyArea("") != otherArea {
		t.Fatal("area mapping")
	}
}

func TestRowsPreview(t *testing.T) {
	art := testArtwork(t)
	examples := []struct {
		label, value, icon string
		on                 bool
		artwork            string
	}{
		{"Mic", "Live", "mic-mute", true, ""},
		{"Mic", "Muted", "mic-mute-muted", true, ""},
		{"Echo strength", "Strong", "echo", false, ""},
		{"Processing", "Error", "mode-element", false, ""},
		{"Playback", "On", "speaker-mute", true, ""},
		{"Record", "Recording", "record-stop", true, ""},
		{"Record PC", "Off", "record-computer", false, ""},
		{"Airhorn", "Playing", "soundboard-play", false, art},
		{"Lofi beats", "Paused", "media-play", false, art},
		{"Google Chrome", "42%", "app-audio", false, ""},
		{"Concentrate", "Active", "hue-scene", false, ""},
		{"Soundboard", "", "deck-page", false, ""},
		{"Sync", "Wait", "huesync-sync", false, ""},
		{"Up", "1/2", "deck-up", false, ""},
	}
	const columns, gap = 7, 16
	rows := (len(examples) + columns - 1) / columns
	sheet := image.NewRGBA(image.Rect(0, 0, columns*112+(columns-1)*gap, rows*112+(rows-1)*gap))
	fill(sheet, 0, 0, sheet.Bounds().Dx(), sheet.Bounds().Dy(), color.RGBA{11, 12, 13, 255})
	for n, e := range examples {
		im := decodeKey(t, e.label, e.value, e.icon, e.on, e.artwork)
		x0, y0 := (n%columns)*(112+gap), (n/columns)*(112+gap)
		for y := 0; y < 112; y++ {
			for x := 0; x < 112; x++ {
				// The keycap's rounded corner, as it appears on the device.
				if !insideRounded(float64(x)+0.5, float64(y)+0.5, 0, 0, 112, 112, frameRadius) {
					continue
				}
				sheet.Set(x0+x, y0+y, im.At(x, y))
			}
		}
	}
	writePreview(t, os.Getenv("SNOOFER_ROWS_PREVIEW"), sheet)
}
