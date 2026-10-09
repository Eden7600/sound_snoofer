package streamdeck

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"testing"

	"sound-snoofer/internal/placeholder"
)

// unrotate decodes a key JPEG and undoes the hardware rotation.
func unrotate(t *testing.T, raw []byte) *image.RGBA {
	t.Helper()
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

// pieArtwork is a scene-style disc of three wedges on a transparent square.
func pieArtwork(t *testing.T) string {
	t.Helper()
	im := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	wedges := []color.NRGBA{{142, 197, 255, 255}, {244, 241, 232, 255}, {92, 141, 219, 255}}
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			dx, dy := float64(x)-31.5, float64(y)-31.5
			if dx*dx+dy*dy > 31*31 {
				continue
			}
			turn := (math.Atan2(dy, dx) + math.Pi) / (2 * math.Pi)
			im.SetNRGBA(x, y, wedges[min(2, int(turn*3))])
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

func TestPieBandHasNoRoundEdge(t *testing.T) {
	im := decodeKey(t, "Concentrate", "Ready", "hue-scene", false, pieArtwork(t))
	// The band's corners and edges show wedges, not the dark key.
	for _, p := range []image.Point{{1, 1}, {110, 1}, {1, 54}, {110, 54}, {56, 1}, {56, 54}} {
		if near(im.RGBAAt(p.X, p.Y), keyColor, 40) {
			t.Fatalf("band shows background at %v", p)
		}
	}
}

func TestSceneAndClipKeysOmitRoutineWords(t *testing.T) {
	// A playing clip drops its status word: the artwork grows past the
	// usual band, and the label sits in the bottom rows on the green fill.
	playing := decodeKey(t, "Airhorn", "Playing", "soundboard-play", false, testArtwork(t))
	if got := playing.RGBAAt(30, 70); near(got, soundboardArea, 40) {
		t.Fatal("artwork band did not grow into the freed space", got)
	}
	ink := false
	for y := 88; y < 104 && !ink; y++ {
		for x := 10; x < 60 && !ink; x++ {
			ink = near(playing.RGBAAt(x, y), inkColor, 60)
		}
	}
	if !ink || !near(playing.RGBAAt(100, 96), soundboardArea, 40) {
		t.Fatal("label not at the bottom of the green key")
	}
	if !routineWord("hue-scene", "ACTIVE") || routineWord("hue-scene", "WAIT") || routineWord("mic-mute", "READY") {
		t.Fatal("routine word rules")
	}
	// Wait still shows on a scene key.
	waiting := decodeKey(t, "Relax", "Wait", "hue-scene", false, "")
	amber := false
	for y := 76; y < 104 && !amber; y++ {
		for x := 10; x < 104 && !amber; x++ {
			amber = near(waiting.RGBAAt(x, y), attentionColor, 40)
		}
	}
	if !amber {
		t.Fatal("pending scene lost Wait")
	}
}

func TestLongValuesWrapInPlaceOfTheIcon(t *testing.T) {
	set := typefaces()
	room := layoutValue(set, "UPSTAIRS LIVING ROOM")
	if len(room.lines) != 2 || room.icon || room.top != 48 {
		t.Fatal("room name should wrap in place of the icon", room)
	}
	short := layoutValue(set, "ON")
	if len(short.lines) != 1 || !short.icon || short.px != 30 {
		t.Fatal("short value changed", short)
	}
	// Two lines of value stay clear of the label rows and the bottom inset.
	if room.top < labelTop+2*17 || room.top+len(room.lines)*(room.px+1) > 104 {
		t.Fatal("wrapped value leaves its rows", room)
	}
}

// clipArtwork is a wide image fitted into the 64 px square the soundboard
// sends: transparent bands above and below, the picture between.
func clipArtwork(t *testing.T) string {
	t.Helper()
	im := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 16; y < 48; y++ {
		for x := 0; x < 64; x++ {
			c := color.NRGBA{224, 98, 58, 255}
			if (x-20)*(x-20)+(y-32)*(y-32) < 120 {
				c = color.NRGBA{247, 215, 116, 255}
			}
			im.SetNRGBA(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

func TestArtworkByKind(t *testing.T) {
	if artworkKind("hue-scene") != pieArt || artworkKind("soundboard-play") != clipArt || artworkKind("app-audio") != programArt ||
		artworkKind("app-audio-muted") != programArt || artworkKind("media-play") != coverArt {
		t.Fatal("artwork kinds")
	}
	// Clip art is shown whole: its transparent bands show the key's fill,
	// and the picture is not zoomed past the band's edges.
	clip := decodeKey(t, "Airhorn", "Ready", "soundboard-play", false, clipArtwork(t))
	if !near(clip.RGBAAt(56, 4), keyColor, 30) {
		t.Fatal("clip art was cropped to fill the band", clip.RGBAAt(56, 4))
	}
	if near(clip.RGBAAt(56, 42), keyColor, 30) {
		t.Fatal("clip art missing")
	}
	// A program icon sits in the icon slot, between the name and the value.
	app := decodeKey(t, "Discord", "42%", "app-audio", false, placeholder.Art("Discord", placeholder.App))
	if near(app.RGBAAt(26, 58), keyColor, 30) {
		t.Fatal("program icon missing from the icon slot")
	}
	if !near(app.RGBAAt(90, 58), keyColor, 30) {
		t.Fatal("program icon spread past the icon slot", app.RGBAAt(90, 58))
	}
}

func TestRowsPreview(t *testing.T) {
	art := testArtwork(t)
	pie := pieArtwork(t)
	clip := clipArtwork(t)
	app := placeholder.Art("Google Chrome", placeholder.App)
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
		{"Airhorn", "Playing", "soundboard-play", false, clip},
		{"Lofi beats", "Paused", "media-play", false, art},
		{"Google Chrome", "42%", "app-audio", false, app},
		{"Concentrate", "Active", "hue-scene", false, pie},
		{"Soundboard", "", "deck-page", false, ""},
		{"Sync", "Wait", "huesync-sync", false, ""},
		{"Up", "1/2", "deck-up", false, ""},
		{"Room", "Upstairs Living Room", "", false, ""},
		{"Mute", "Inactive", "mic-mute", false, ""}, // Drawn inactive below.
	}
	const columns, gap = 7, 16
	rows := (len(examples) + columns - 1) / columns
	sheet := image.NewRGBA(image.Rect(0, 0, columns*112+(columns-1)*gap, rows*112+(rows-1)*gap))
	fill(sheet, 0, 0, sheet.Bounds().Dx(), sheet.Bounds().Dy(), color.RGBA{11, 12, 13, 255})
	for n, e := range examples {
		im := decodeKey(t, e.label, e.value, e.icon, e.on, e.artwork)
		if e.value == "Inactive" {
			raw, _ := drawKey([]string{e.label, "On"}, 112, 112, true, e.icon, false, true, "", 0)
			im = unrotate(t, raw)
		}
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

func TestInactiveKeysAndDialsAreGrey(t *testing.T) {
	raw, _ := drawKey([]string{"Mute", "On"}, 112, 112, true, "mic-mute", false, true, "", 0)
	im, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	// Dark, not filled cyan, and nothing in the value row's place but the
	// enlarged grey icon.
	if r, g, b, _ := im.At(60, 11).RGBA(); !near(rgba(r, g, b), keyColor, 16) {
		t.Fatal("inactive key is filled")
	}
	grey, colour := false, false
	for y := 0; y < 112; y++ {
		for x := 0; x < 112; x++ {
			r, g, b, _ := im.At(x, y).RGBA()
			c := rgba(r, g, b)
			grey = grey || near(c, unknownColor, 20)
			colour = colour || near(c, micArea, 60)
		}
	}
	if !grey || colour {
		t.Fatal("inactive key is not grey", grey, colour)
	}
	var knobs [Encoders]knobPresentation
	knobs[0] = knobPresentation{Target: "Mic", Value: "0.0 dB", Icon: "mic-mute", Meter: true, LevelKnown: true, LevelDB: -10, PositionKnown: true, Position: 0.8, Inactive: true}
	strip, _ := renderKnobs(knobs, 0)
	sim, err := jpeg.Decode(bytes.NewReader(strip))
	if err != nil {
		t.Fatal(err)
	}
	// No meter along the bottom edge (strip rotated: panel (x, y) is at (y, 1199-x)).
	if r, g, b, _ := sim.At(95, 1199-10).RGBA(); near(rgba(r, g, b), meterGreenColor, 40) {
		t.Fatal("inactive dial shows a meter")
	}
}
