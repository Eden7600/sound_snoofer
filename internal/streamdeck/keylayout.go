package streamdeck

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"math"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
)

// Key rows (openspec/changes/deck-rows-left): label, icon and value each own
// a full-width row, flush left, inside an 8 px inset the keycap never hides.
const (
	keyInset    = 10 // Left edge of text and icons.
	labelTop    = 8
	iconTop     = 42
	iconSize    = 34
	valueTop    = 76
	bandHeight  = 56 // Artwork band at the top of an artwork key.
	frameWidth  = 4  // Error frame stroke.
	frameRadius = 10 // Assumed keycap corner radius.
)

// keyStyle is how a state paints a key.
type keyStyle struct {
	background, label, icon, value color.RGBA
	frame, dashes, alert           bool // Error frame, pending dashes, "!" in place of the icon.
}

// styleFor maps the semantic accent to the key's surfaces: on states fill
// with the area or state colour; others stay dark with area-coloured text.
func styleFor(accent color.RGBA, value string, area color.RGBA) keyStyle {
	switch accent {
	case activeColor:
		return keyStyle{background: area, label: inkColor, icon: inkColor, value: inkColor}
	case criticalColor:
		if value == "ERROR" || value == "FAILED" {
			return keyStyle{background: keyColor, label: area, icon: criticalColor, value: criticalColor, frame: true, alert: true}
		}
		return keyStyle{background: criticalColor, label: inkColor, icon: inkColor, value: inkColor}
	case attentionColor:
		if value == "WAIT" {
			return keyStyle{background: keyColor, label: area, icon: attentionColor, value: attentionColor, dashes: true}
		}
		return keyStyle{background: attentionColor, label: inkColor, icon: inkColor, value: inkColor}
	}
	switch value {
	case "N/A", "UNKNOWN", "UNAVAILABLE":
		return keyStyle{background: keyColor, label: area, icon: unknownColor, value: unknownColor}
	case "OFF", "READY", "PAUSED", "MISSING":
		return keyStyle{background: keyColor, label: area, icon: area, value: offValue}
	}
	return keyStyle{background: keyColor, label: area, icon: area, value: lightValue}
}

// renderArtwork draws one key as a rotated JPEG, with scrolling text at its
// start. lines are the label and value; artwork, when it decodes, fills a
// band above the label.
func renderArtwork(lines []string, w, h int, on bool, icon string, fallback bool, artwork string) []byte {
	key, _ := drawKey(lines, w, h, on, icon, fallback, false, artwork, 0)
	return key
}

// drawKey is renderArtwork at a scroll phase, greyed without a value when
// inactive; it reports whether any of the key's text scrolls.
func drawKey(lines []string, w, h int, on bool, icon string, fallback, inactive bool, artwork string, phase time.Duration) ([]byte, bool) {
	scrolls := false
	// Supersample geometry and text, then average down.
	const scale = 4
	im := image.NewRGBA(image.Rect(0, 0, w*scale, h*scale))

	label, value := "", ""
	if len(lines) > 0 {
		label = lines[0]
	}
	if len(lines) > 1 {
		value = strings.ToUpper(lines[1])
	}
	accent := keyAccent(value, icon, on, fallback)
	switch value {
	case "RECORDING":
		value = "REC"
	case "STOPPED":
		value = "READY"
	case "AUDIBLE":
		value = "ON"
	case "PENDING":
		value = "WAIT"
	case "UNAVAIL":
		value = "N/A"
	}
	if fallback && accent == attentionColor && value != "WAIT" && value != "" {
		value += "*"
	}
	if routineWord(icon, value) {
		value = "" // The fill already says Playing or Active.
	}
	style := styleFor(accent, value, keyArea(icon))
	if inactive {
		// No effect right now: greyed in place, with no state to show.
		style = keyStyle{background: keyColor, label: unknownColor, icon: unknownColor, value: unknownColor}
		value, artwork = "", ""
	}
	fill(im, 0, 0, w*scale, h*scale, style.background)

	set := typefaces()
	art, hasArt := decodeArtwork(artwork)
	kind := artworkKind(icon)
	if hasArt {
		switch {
		case kind == programArt:
			// A program icon sits icon-sized above a one-line title.
			drawArtIcon(im, scale, art, keyInset, 12, 40)
			scrolls = drawArtworkRows(im, scale, label, value, style, phase)
		case value == "":
			scrolls = drawArtworkLabel(im, scale, w, art, kind, label, style.label, phase)
		default:
			drawBand(im, scale, w, bandHeight, art, kind)
			scrolls = drawArtworkRows(im, scale, label, value, style, phase)
		}
	} else {
		if label != "" {
			if f, ok := set.fitWords(labelFace, label, keyTextWidth, 16, 12, 2); ok {
				for n, line := range f.lines {
					set.draw(im, scale, labelFace, f.px, keyInset, labelTop+n*(f.px+1), line, style.label)
				}
			} else {
				set.drawScrolling(im, scale, labelFace, 15, keyInset, labelTop, keyTextWidth, label, style.label, phase)
				scrolls = true
			}
		}
		v := layoutValue(set, value)
		top, size := iconTop, iconSize
		if value == "" {
			top, size = 56, 40 // The value row's space goes to the icon.
		}
		switch {
		case style.alert:
			drawAlert(im, scale, keyInset, top, size)
		case v.icon:
			drawIconBox(im, scale, icon, style.icon, keyInset, top, size)
		}
		if v.scroll {
			set.drawScrolling(im, scale, valueFace, v.px, keyInset, v.top, keyTextWidth, value, style.value, phase)
			scrolls = true
		}
		for n, line := range v.lines {
			set.draw(im, scale, valueFace, v.px, keyInset, v.top+n*(v.px+1), line, style.value)
		}
	}
	if style.frame {
		drawFrame(im, scale, w, h, criticalColor)
	}
	if style.dashes {
		for x := 0; x < w; x += 14 {
			fill(im, x*scale, 0, min(x+8, w)*scale, 5*scale, attentionColor)
		}
	}
	return encodeKey(im, scale, w, h), scrolls
}

// valueLayout is where a key's value goes, and whether the icon keeps its
// row.
type valueLayout struct {
	lines        []string
	px, top      int
	icon, scroll bool // scroll: one line that scrolls, drawn in place of lines.
}

// layoutValue sets a value on one line at 30 to 22 px; a value of several
// words that does not fit wraps to two lines at 22 to 16 px in place of the
// icon. Anything longer scrolls on one line at 20 px.
func layoutValue(set *typeset, value string) valueLayout {
	if value == "" {
		return valueLayout{icon: true}
	}
	if f, ok := set.fitWords(valueFace, value, keyTextWidth, 30, 22, 1); ok {
		return valueLayout{lines: f.lines, px: f.px, top: valueTop, icon: true}
	}
	if strings.Contains(strings.TrimSpace(value), " ") {
		if f, ok := set.fitWords(valueFace, value, keyTextWidth, 22, 16, 2); ok {
			if len(f.lines) == 1 {
				return valueLayout{lines: f.lines, px: f.px, top: valueTop, icon: true}
			}
			return valueLayout{lines: f.lines, px: f.px, top: 48}
		}
	}
	return valueLayout{px: 20, top: valueTop + 4, icon: true, scroll: true}
}

// encodeKey averages the supersampled canvas and rotates it 90 degrees
// counterclockwise for the + XL.
func encodeKey(im *image.RGBA, scale, w, h int) []byte {
	rotated := image.NewRGBA(image.Rect(0, 0, h, w))
	samples := uint32(scale * scale)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b uint32
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					p := im.RGBAAt(x*scale+dx, y*scale+dy)
					r += uint32(p.R)
					g += uint32(p.G)
					b += uint32(p.B)
				}
			}
			rotated.SetRGBA(y, w-1-x, color.RGBA{uint8(r / samples), uint8(g / samples), uint8(b / samples), 255})
		}
	}
	var out bytes.Buffer
	jpeg.Encode(&out, rotated, &jpeg.Options{Quality: 92})
	return out.Bytes()
}

// drawIconBox draws a code-drawn icon into a size-pixel square at key
// pixel (x, y), averaging its 80 px drawing area down so strokes keep their
// weight as coverage.
func drawIconBox(im *image.RGBA, scale int, icon string, ink color.RGBA, x, y, size int) {
	glyph := image.NewRGBA(image.Rect(0, 0, 112*scale, 112*scale))
	if !drawIcon(glyph, icon, ink) {
		return
	}
	source := image.Rect(16*scale, 12*scale, 96*scale, 92*scale)
	target := size * scale
	for ty := 0; ty < target; ty++ {
		sy0 := source.Min.Y + ty*source.Dy()/target
		sy1 := max(sy0+1, source.Min.Y+(ty+1)*source.Dy()/target)
		for tx := 0; tx < target; tx++ {
			sx0 := source.Min.X + tx*source.Dx()/target
			sx1 := max(sx0+1, source.Min.X+(tx+1)*source.Dx()/target)
			covered, total := 0, 0
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					total++
					if glyph.RGBAAt(sx, sy).A != 0 {
						covered++
					}
				}
			}
			if covered == 0 {
				continue
			}
			px, py := x*scale+tx, y*scale+ty
			im.SetRGBA(px, py, blend(im.RGBAAt(px, py), ink, float64(covered)/float64(total)))
		}
	}
}

// blend mixes ink over base by coverage t.
func blend(base, ink color.RGBA, t float64) color.RGBA {
	mix := func(a, b uint8) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*t + 0.5) }
	return color.RGBA{mix(base.R, ink.R), mix(base.G, ink.G), mix(base.B, ink.B), 255}
}

// drawAlert is the error mark: a red square with "!" in place of the icon.
func drawAlert(im *image.RGBA, scale, x, y, size int) {
	fill(im, x*scale, y*scale, (x+size)*scale, (y+size)*scale, criticalColor)
	set := typefaces()
	px := size * 4 / 5
	mark := set.width(valueFace, px, "!")
	set.draw(im, scale, valueFace, px, x+(size-mark)/2, y+(size-px)/2-1, "!", keyColor)
}

// drawFrame strokes a rounded rectangle just inside the key's edge, so the
// rounded keycap does not clip it unevenly.
func drawFrame(im *image.RGBA, scale, w, h int, c color.RGBA) {
	outer := float64(frameRadius * scale)
	inner := float64((frameRadius - frameWidth) * scale)
	stroke := float64(frameWidth * scale)
	W, H := float64(w*scale), float64(h*scale)
	for py := 0; py < h*scale; py++ {
		for px := 0; px < w*scale; px++ {
			x, y := float64(px)+0.5, float64(py)+0.5
			if insideRounded(x, y, 0, 0, W, H, outer) && !insideRounded(x, y, stroke, stroke, W-stroke, H-stroke, inner) {
				im.SetRGBA(px, py, c)
			}
		}
	}
}

// insideRounded reports whether (x, y) lies in the rectangle with corner
// radius r.
func insideRounded(x, y, x0, y0, x1, y1, r float64) bool {
	if x < x0 || x >= x1 || y < y0 || y >= y1 {
		return false
	}
	cx := min(max(x, x0+r), x1-r)
	cy := min(max(y, y0+r), y1-r)
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}

// routineWord reports a status word that scene and clip keys leave to
// their fill: Ready, Active and Playing.
func routineWord(icon, value string) bool {
	switch strings.TrimSuffix(icon, "-off") {
	case "hue-scene", "soundboard-play":
		return value == "READY" || value == "ACTIVE" || value == "PLAYING"
	}
	return false
}

// drawArtworkRows sets an artwork key's text beneath its artwork: the label
// on one line at y 60, the value at y 80, each scrolling when it cannot fit.
// It reports whether either scrolls.
func drawArtworkRows(im *image.RGBA, scale int, label, value string, style keyStyle, phase time.Duration) bool {
	set := typefaces()
	scrolls := false
	if label != "" {
		if f, ok := set.fitWords(labelFace, label, keyTextWidth, 15, 12, 1); ok {
			set.draw(im, scale, labelFace, f.px, keyInset, bandHeight+4, f.lines[0], style.label)
		} else {
			set.drawScrolling(im, scale, labelFace, 15, keyInset, bandHeight+4, keyTextWidth, label, style.label, phase)
			scrolls = true
		}
	}
	if value != "" {
		if f, ok := set.fitWords(valueFace, value, keyTextWidth, 24, 16, 1); ok {
			set.draw(im, scale, valueFace, f.px, keyInset, bandHeight+24, f.lines[0], style.value)
		} else {
			set.drawScrolling(im, scale, valueFace, 18, keyInset, bandHeight+26, keyTextWidth, value, style.value, phase)
			scrolls = true
		}
	}
	return scrolls
}

// drawArtworkLabel lays out an artwork key with no value: the label sits at
// the bottom on up to two lines, and the artwork fills the space above it.
// It reports whether the label scrolls.
func drawArtworkLabel(im *image.RGBA, scale, w int, art image.Image, kind artKind, label string, ink color.RGBA, phase time.Duration) bool {
	set := typefaces()
	if label == "" {
		drawBand(im, scale, w, 104, art, kind)
		return false
	}
	f, fits := set.fitWords(labelFace, label, keyTextWidth, 15, 12, 2)
	if !fits {
		f = fitted{px: 15, lines: []string{label}}
	}
	top := 104 - len(f.lines)*(f.px+1)
	drawBand(im, scale, w, top-4, art, kind)
	if !fits {
		set.drawScrolling(im, scale, labelFace, 15, keyInset, top, keyTextWidth, label, ink, phase)
		return true
	}
	for n, line := range f.lines {
		set.draw(im, scale, labelFace, f.px, keyInset, top+n*(f.px+1), line, ink)
	}
	return false
}

// drawBand scales artwork into a band height pixels tall at the top of the
// key, by content type. A cover is cropped to the band's shape; a pie shows
// the rectangle inscribed in its circle, so no round edge shows; clip art is
// shown whole, scaled to the band's height and centred over the fill.
func drawBand(im *image.RGBA, scale, w, height int, art image.Image, kind artKind) {
	b := art.Bounds()
	if kind == clipArt {
		side := min(w, height)
		x := (w - side) / 2
		xdraw.CatmullRom.Scale(im, image.Rect(x*scale, (height-side)*scale, (x+side)*scale, height*scale), art, b, draw.Over, nil)
		return
	}
	aspect := float64(w) / float64(height)
	cx := float64(b.Min.X) + float64(b.Dx())/2
	cy := float64(b.Min.Y) + float64(b.Dy())/2
	side := float64(min(b.Dx(), b.Dy()))
	cropW, cropH := side, side/aspect
	if aspect < 1 {
		cropW, cropH = side*aspect, side
	}
	if kind == pieArt {
		r := side/2 - 1
		cropH = 2 * r / math.Sqrt(1+aspect*aspect)
		cropW = aspect * cropH
	}
	source := image.Rect(int(math.Ceil(cx-cropW/2)), int(math.Ceil(cy-cropH/2)), int(math.Floor(cx+cropW/2)), int(math.Floor(cy+cropH/2)))
	xdraw.CatmullRom.Scale(im, image.Rect(0, 0, w*scale, height*scale), art, source, draw.Over, nil)
}

// artKind is what a key's artwork shows, which decides how it is drawn
// (openspec/changes/deck-artwork-kinds).
type artKind int

const (
	coverArt   artKind = iota // Album art: fills a cropped band.
	pieArt                    // Scene wedges: cropped inside the circle.
	clipArt                   // A soundboard image: shown whole.
	programArt                // A program icon or placeholder: the key's icon.
)

// artworkKind names the artwork's content type from the control's icon.
func artworkKind(icon string) artKind {
	icon = strings.TrimSuffix(strings.TrimSuffix(icon, "-muted"), "-off")
	switch {
	case icon == "hue-scene":
		return pieArt
	case strings.HasPrefix(icon, "soundboard-"):
		return clipArt
	case icon == "app-audio":
		return programArt
	}
	return coverArt
}

// drawArtIcon scales artwork smoothly into a size-pixel square at key pixel
// (x, y), as the key's icon.
func drawArtIcon(im *image.RGBA, scale int, art image.Image, x, y, size int) {
	xdraw.CatmullRom.Scale(im, image.Rect(x*scale, y*scale, (x+size)*scale, (y+size)*scale), art, art.Bounds(), draw.Over, nil)
}
