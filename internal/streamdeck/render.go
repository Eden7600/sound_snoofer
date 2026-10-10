package streamdeck

import (
	"bytes"
	"encoding/base64"

	"image"
	"image/color"
	"image/png"
	"strings"
	"time"
)

// Five-column glyphs keep the device display independent of installed fonts.
var glyphs = map[rune][5]byte{
	'%': {35, 19, 8, 100, 98},
	'A': {126, 9, 9, 9, 126}, 'B': {127, 73, 73, 73, 54}, 'C': {62, 65, 65, 65, 34}, 'D': {127, 65, 65, 34, 28}, 'E': {127, 73, 73, 73, 65}, 'F': {127, 9, 9, 9, 1}, 'G': {62, 65, 73, 73, 122}, 'H': {127, 8, 8, 8, 127}, 'I': {65, 65, 127, 65, 65}, 'J': {32, 64, 65, 63, 1}, 'K': {127, 8, 20, 34, 65}, 'L': {127, 64, 64, 64, 64}, 'M': {127, 2, 12, 2, 127}, 'N': {127, 4, 8, 16, 127}, 'O': {62, 65, 65, 65, 62}, 'P': {127, 9, 9, 9, 6}, 'Q': {62, 65, 81, 33, 94}, 'R': {127, 9, 25, 41, 70}, 'S': {38, 73, 73, 73, 50}, 'T': {1, 1, 127, 1, 1}, 'U': {63, 64, 64, 64, 63}, 'V': {31, 32, 64, 32, 31}, 'W': {63, 64, 56, 64, 63}, 'X': {99, 20, 8, 20, 99}, 'Y': {7, 8, 112, 8, 7}, 'Z': {97, 81, 73, 69, 67},
	'0': {62, 81, 73, 69, 62}, '1': {0, 66, 127, 64, 0}, '2': {66, 97, 81, 73, 70}, '3': {33, 65, 69, 75, 49}, '4': {24, 20, 18, 127, 16}, '5': {39, 69, 69, 69, 57}, '6': {60, 74, 73, 73, 48}, '7': {1, 113, 9, 5, 3}, '8': {54, 73, 73, 73, 54}, '9': {6, 73, 73, 41, 30}, '-': {8, 8, 8, 8, 8}, '.': {0, 96, 96, 0, 0}, '/': {32, 16, 8, 4, 2}, '?': {2, 1, 81, 9, 6}, '+': {8, 8, 62, 8, 8}, ':': {0, 54, 54, 0, 0}, '*': {20, 8, 62, 8, 20}}

func text(im *image.RGBA, x, y, scale int, s string, c color.RGBA) {
	for _, r := range strings.ToUpper(s) {
		g := glyphs[r]
		for col, v := range g {
			for row := 0; row < 7; row++ {
				if v&(1<<row) != 0 {
					for dx := 0; dx < scale; dx++ {
						for dy := 0; dy < scale; dy++ {
							im.SetRGBA(x+col*scale+dx, y+row*scale+dy, c)
						}
					}
				}
			}
		}
		x += 6 * scale
	}
}
func render(lines []string, w, h int, on bool, icon string, fallback bool) []byte {
	return renderArtwork(lines, w, h, on, icon, fallback, "")
}

type keyPresentation struct {
	Artwork            string
	Label, Value, Icon string
	Fallback, Inactive bool
	Brightness         float64 // Overall brightness for the badge on artwork keys, when known.
	BrightnessKnown    bool
}
type knobPresentation struct {
	Target, Value, Name, Status string
	Meter, LevelKnown           bool
	LevelDB                     float64
	PeakKnown, PositionKnown    bool
	PeakDB, Position, ZeroMark  float64
	Timers                      [2]TimerTile
	Artwork                     string // Shown left of the dial's text when set.
	Icon                        string // The control's icon; names its area colour.
	Inactive                    bool   // Greyed, without position track or meter.
}
type presentation struct {
	Keys  [Keys]keyPresentation
	Knobs [Encoders]knobPresentation
}

// renderKey draws a key at a scroll phase and reports whether its text
// scrolls.
func renderKey(key keyPresentation, phase time.Duration) ([]byte, bool) {
	var lines []string
	if key.Label != "" {
		lines = []string{key.Label, key.Value}
	}
	spec := keySpec{lines: lines, w: 112, h: 112, on: key.Value == "On" || key.Value == "Recording", icon: key.Icon,
		fallback: key.Fallback, inactive: key.Inactive, artwork: key.Artwork}
	if key.BrightnessKnown {
		spec.brightness = &key.Brightness
	}
	return drawKeySpec(spec, phase)
}

func renderPresentation(view presentation) ([][]byte, []byte) {
	tiles := make([][]byte, Keys)
	for n, key := range view.Keys {
		tiles[n], _ = renderKey(key, 0)
	}
	touch, _ := renderKnobs(view.Knobs, 0)
	return tiles, touch
}

func fill(im *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			im.SetRGBA(x, y, c)
		}
	}
}

// The artwork contract: a square PNG at most maxArtworkSize pixels, base64
// at most maxArtworkBytes. Scenes and covers use the full size so they stay
// sharp when cropped and scaled to a key's width.
const (
	maxArtworkSize  = 128
	maxArtworkBytes = 128 * 1024
)

// decodeArtwork accepts only the bounded thumbnail contract, never source files.
func decodeArtwork(artwork string) (image.Image, bool) {
	if artwork == "" || len(artwork) > maxArtworkBytes {
		return nil, false
	}
	data, err := base64.StdEncoding.DecodeString(artwork)
	if err != nil {
		return nil, false
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Width != cfg.Height || cfg.Width > maxArtworkSize {
		return nil, false
	}
	source, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	return source, true
}
