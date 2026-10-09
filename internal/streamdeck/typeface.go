package streamdeck

import (
	_ "embed"
	"image"
	"image/color"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Archivo (SIL Open Font License, fonts/OFL.txt): Bold for labels and
// Condensed ExtraBold for values.
var (
	//go:embed fonts/Archivo-Bold.ttf
	labelFontData []byte
	//go:embed fonts/ArchivoCondensed-ExtraBold.ttf
	valueFontData []byte
)

// keyTextWidth is the width text may use on a key: x 10 to 104, inside the
// rounded keycap's corners.
const keyTextWidth = 94

type typeface int

const (
	labelFace typeface = iota
	valueFace
)

// typeset holds the parsed faces. Missing fonts fall back to the bitmap
// glyphs, so a font that fails to parse never blanks the deck.
type typeset struct {
	fonts [2]*opentype.Font
}

var typefaces = sync.OnceValue(func() *typeset {
	set := &typeset{}
	for n, data := range [][]byte{labelFontData, valueFontData} {
		parsed, err := opentype.Parse(data)
		if err != nil {
			return &typeset{}
		}
		set.fonts[n] = parsed
	}
	return set
})

// face returns a face at px, or false for the bitmap fallback. Faces are
// made per use because they are not safe for concurrent use.
func (t *typeset) face(kind typeface, px int) (font.Face, bool) {
	parsed := t.fonts[kind]
	if parsed == nil {
		return nil, false
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(px), DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, false
	}
	return face, true
}

// bitmapScale is the glyph multiple that best matches a type size.
func bitmapScale(px int) int {
	return max(1, (px+4)/8)
}

// width is s's advance in key pixels at px.
func (t *typeset) width(kind typeface, px int, s string) int {
	face, ok := t.face(kind, px)
	if !ok {
		return len([]rune(s)) * 6 * bitmapScale(px)
	}
	defer face.Close()
	return font.MeasureString(face, s).Ceil()
}

// draw sets s with the top of its line box at key pixel (x, y) on a canvas
// supersampled by scale.
func (t *typeset) draw(im *image.RGBA, scale int, kind typeface, px, x, y int, s string, c color.RGBA) {
	face, ok := t.face(kind, px*scale)
	if !ok {
		size := bitmapScale(px)
		text(im, x*scale, (y+max(0, px-7*size)/2)*scale, size*scale, s, c)
		return
	}
	defer face.Close()
	d := font.Drawer{Dst: im, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x*scale, y*scale+face.Metrics().Ascent.Ceil())}
	d.DrawString(s)
}

// fitted is text set to fit a box: its size, its lines, and whether it had
// to be cut.
type fitted struct {
	px    int
	lines []string
	cut   bool
}

// fit sets s within width on at most maxLines lines. Each size from maxPx
// down to minPx tries one line, then word wrapping. Past that it breaks
// inside words at minPx, and cuts with an ellipsis only as a last resort.
func (t *typeset) fit(kind typeface, s string, width, maxPx, minPx, maxLines int) fitted {
	s = strings.Join(strings.Fields(s), " ")
	for px := maxPx; px >= minPx; px-- {
		if t.width(kind, px, s) <= width {
			return fitted{px: px, lines: []string{s}}
		}
		if lines, ok := t.wrap(kind, px, s, width, false); ok && len(lines) <= maxLines {
			return fitted{px: px, lines: lines}
		}
	}
	lines, _ := t.wrap(kind, minPx, s, width, true)
	if len(lines) <= maxLines {
		return fitted{px: minPx, lines: lines}
	}
	rest := strings.Join(lines[maxLines-1:], " ")
	lines = append(lines[:maxLines-1:maxLines-1], t.ellipsize(kind, minPx, rest, width))
	return fitted{px: minPx, lines: lines, cut: true}
}

// wrap breaks s into lines no wider than width, at spaces, or anywhere when
// breakWords is set. It reports false when a word alone is too wide.
func (t *typeset) wrap(kind typeface, px int, s string, width int, breakWords bool) ([]string, bool) {
	var lines []string
	line := ""
	for _, word := range strings.Fields(s) {
		candidate := word
		if line != "" {
			candidate = line + " " + word
		}
		if t.width(kind, px, candidate) <= width {
			line = candidate
			continue
		}
		if line != "" {
			lines = append(lines, line)
			line = ""
		}
		if t.width(kind, px, word) <= width {
			line = word
			continue
		}
		if !breakWords {
			return nil, false
		}
		for _, r := range word {
			if line != "" && t.width(kind, px, line+string(r)) > width {
				lines = append(lines, line)
				line = ""
			}
			line += string(r)
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines, true
}

// ellipsize shortens s until it and a trailing ellipsis fit width.
func (t *typeset) ellipsize(kind typeface, px int, s string, width int) string {
	mark := "…"
	if t.fonts[kind] == nil {
		mark = "..." // The bitmap glyphs have no ellipsis.
	}
	runes := []rune(s)
	for len(runes) > 0 && t.width(kind, px, string(runes)+mark) > width {
		runes = runes[:len(runes)-1]
	}
	return strings.TrimRight(string(runes), " ") + mark
}
