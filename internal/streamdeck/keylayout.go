package streamdeck

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"strings"
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

// renderArtwork draws one key as a rotated JPEG. lines are the label and
// value; artwork, when it decodes, fills a band above the label.
func renderArtwork(lines []string, w, h int, on bool, icon string, fallback bool, artwork string) []byte {
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
	style := styleFor(accent, value, keyArea(icon))
	fill(im, 0, 0, w*scale, h*scale, style.background)

	set := typefaces()
	if art, ok := decodeArtwork(artwork); ok {
		drawBand(im, scale, w, art)
		if label != "" {
			f := set.fit(labelFace, label, keyTextWidth, 15, 12, 1)
			set.draw(im, scale, labelFace, f.px, keyInset, bandHeight+4, f.lines[0], style.label)
		}
		if value != "" {
			f := set.fit(valueFace, value, keyTextWidth, 24, 16, 1)
			set.draw(im, scale, valueFace, f.px, keyInset, bandHeight+24, f.lines[0], style.value)
		}
	} else {
		if label != "" {
			f := set.fit(labelFace, label, keyTextWidth, 16, 12, 2)
			for n, line := range f.lines {
				set.draw(im, scale, labelFace, f.px, keyInset, labelTop+n*(f.px+1), line, style.label)
			}
		}
		top, size := iconTop, iconSize
		if value == "" {
			top, size = 56, 40 // The value row's space goes to the icon.
		}
		if style.alert {
			drawAlert(im, scale, keyInset, top, size)
		} else {
			drawIconBox(im, scale, icon, style.icon, keyInset, top, size)
		}
		if value != "" {
			f := set.fit(valueFace, value, keyTextWidth, 30, 18, 1)
			set.draw(im, scale, valueFace, f.px, keyInset, valueTop, f.lines[0], style.value)
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
	return encodeKey(im, scale, w, h)
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

// drawBand scales square artwork to the key's width and centre-crops it into
// the band at the top.
func drawBand(im *image.RGBA, scale, w int, art image.Image) {
	bounds := art.Bounds()
	width, height := w*scale, bandHeight*scale
	offset := (width - height) / 2 // Rows of the scaled square above the band.
	for y := 0; y < height; y++ {
		sy := bounds.Min.Y + (y+offset)*bounds.Dy()/width
		for x := 0; x < width; x++ {
			sx := bounds.Min.X + x*bounds.Dx()/width
			r, g, b, a := art.At(sx, sy).RGBA()
			if a == 0 {
				continue
			}
			pixel := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255}
			im.SetRGBA(x, y, blend(im.RGBAAt(x, y), unpremultiply(pixel, a), float64(a)/0xffff))
		}
	}
}

// unpremultiply restores straight colour from premultiplied components.
func unpremultiply(c color.RGBA, a uint32) color.RGBA {
	if a == 0 || a == 0xffff {
		return c
	}
	scale := func(v uint8) uint8 { return uint8(min(255, uint32(v)*0xffff/a)) }
	return color.RGBA{scale(c.R), scale(c.G), scale(c.B), 255}
}
