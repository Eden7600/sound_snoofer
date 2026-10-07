// Package placeholder draws stand-in artwork for programs without a logo: a
// tile whose colour is derived from the program's name, so each program has
// its own colour every time and can be spotted at a glance.
package placeholder

import (
	"bytes"
	"encoding/base64"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
)

// Glyph is the symbol drawn on the tile.
type Glyph int

const (
	App   Glyph = iota // A window.
	Media              // A play triangle.
)

const (
	size       = 64 // The control artwork contract's edge.
	saturation = 0.55
	lightness  = 0.45 // Every hue stays readable on the dark deck.
)

// Hue is the name's colour on the colour wheel, 0–360.
func Hue(name string) float64 {
	h := fnv.New32a()
	h.Write([]byte(strings.ToLower(strings.TrimSpace(name))))
	return float64(h.Sum32() % 360)
}

// Art returns a 64 px base64 PNG tile for name with glyph.
func Art(name string, glyph Glyph) string {
	tile := hsl(Hue(name), saturation, lightness)
	ink := color.NRGBA{R: 245, G: 247, B: 250, A: 255}
	im := image.NewNRGBA(image.Rect(0, 0, size, size))
	const inset, radius = 4, 12
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if inRoundedRect(x, y, inset, size-inset, radius) {
				im.SetNRGBA(x, y, tile)
			}
		}
	}
	switch glyph {
	case App:
		// Window: outline, title bar and two dots.
		box := func(x0, y0, x1, y1 int) {
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					im.SetNRGBA(x, y, ink)
				}
			}
		}
		box(17, 19, 47, 22)
		box(17, 43, 47, 46)
		box(17, 19, 20, 46)
		box(44, 19, 47, 46)
		box(17, 26, 47, 28)
		box(22, 22, 24, 25)
		box(26, 22, 28, 25)
	case Media:
		for y := 20; y < 44; y++ {
			half := 12 - int(math.Abs(float64(y-32)))
			for x := 25; x < 25+half*2; x++ {
				im.SetNRGBA(x, y, ink)
			}
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, im) // Encoding an in-memory NRGBA image cannot fail.
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

func inRoundedRect(x, y, lo, hi, r int) bool {
	if x < lo || y < lo || x >= hi || y >= hi {
		return false
	}
	cx := min(max(x, lo+r), hi-1-r)
	cy := min(max(y, lo+r), hi-1-r)
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}

// hsl converts hue (degrees), saturation and lightness (0–1) to an opaque colour.
func hsl(h, s, l float64) color.NRGBA {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g = c, x
	case h < 120:
		r, g = x, c
	case h < 180:
		g, b = c, x
	case h < 240:
		g, b = x, c
	case h < 300:
		r, b = x, c
	default:
		r, b = c, x
	}
	to := func(v float64) uint8 { return uint8(math.Round((v + m) * 255)) }
	return color.NRGBA{R: to(r), G: to(g), B: to(b), A: 255}
}
