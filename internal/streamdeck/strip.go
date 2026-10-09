package streamdeck

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
)

// Touch strip panels (openspec/changes/deck-rows-left): six 200 × 100
// panels in the key typefaces; the last is the page dial.
const (
	panelWidth = 200
	panelInset = 12
	dialWidth  = panelWidth - 2*panelInset // Text and track width.
	trackTop   = 80                        // Position track.
	meterTop   = 90                        // Meter bar, to the panel's bottom.
	meterRows  = 10
)

var (
	trackColor = color.RGBA{38, 41, 44, 255}
	pageColor  = color.RGBA{108, 114, 119, 255} // Neighbouring page names.
)

// renderKnobs draws the strip at a scroll phase and reports whether any of
// its text scrolls.
func renderKnobs(knobs [Encoders]knobPresentation, phase time.Duration) ([]byte, bool) {
	scrolls := false
	touchImage := image.NewRGBA(image.Rect(0, 0, Encoders*panelWidth, 100))
	draw.Draw(touchImage, touchImage.Bounds(), &image.Uniform{keyColor}, image.Point{}, draw.Src)
	for n, knob := range knobs {
		x0 := n * panelWidth
		switch {
		case n == Encoders-1:
			drawPageDial(touchImage, x0, knob)
		case knob.Timers[0] != (TimerTile{}):
			drawTimers(touchImage, x0, knob.Timers)
		case knob.Artwork != "":
			scrolls = drawArtworkKnob(touchImage, x0, knob, phase) || scrolls
		default:
			drawControlKnob(touchImage, x0, knob)
		}
	}
	rotated := image.NewRGBA(image.Rect(0, 0, 100, Encoders*panelWidth))
	for y := 0; y < 100; y++ {
		for x := 0; x < Encoders*panelWidth; x++ {
			rotated.SetRGBA(y, Encoders*panelWidth-1-x, touchImage.RGBAAt(x, y))
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, rotated, &jpeg.Options{Quality: 80})
	return buf.Bytes(), scrolls
}

// drawPageDial shows the previous page, the current page large, and the
// next page, which the page dial carries in Target, Value and Name.
func drawPageDial(im *image.RGBA, x0 int, k knobPresentation) {
	set := typefaces()
	if k.Target != "" {
		f := set.fit(labelFace, "← "+strings.ToUpper(k.Target), dialWidth, 12, 10, 1)
		set.draw(im, 1, labelFace, f.px, x0+panelInset, 10, f.lines[0], pageColor)
	}
	current := set.fit(valueFace, strings.ToUpper(k.Value), dialWidth, 40, 20, 1)
	set.draw(im, 1, valueFace, current.px, x0+panelInset-1, 30, current.lines[0], lightValue)
	if k.Name != "" {
		f := set.fit(labelFace, strings.ToUpper(k.Name)+" →", dialWidth, 12, 10, 1)
		width := set.width(labelFace, f.px, f.lines[0])
		set.draw(im, 1, labelFace, f.px, x0+panelWidth-panelInset-width, 78, f.lines[0], pageColor)
	}
}

// drawControlKnob shows the target in its area colour, a large value with
// its unit set smaller, the position track and the meter.
func drawControlKnob(im *image.RGBA, x0 int, k knobPresentation) {
	set := typefaces()
	x := x0 + panelInset
	if k.Status != "" {
		f := set.fit(labelFace, k.Status, dialWidth/2, 13, 10, 1)
		width := set.width(labelFace, f.px, f.lines[0])
		set.draw(im, 1, labelFace, f.px, x0+panelWidth-panelInset-width, 10, f.lines[0], attentionColor)
	}
	target, ink := keyArea(k.Icon), lightValue
	if k.Inactive {
		target, ink = unknownColor, unknownColor
	}
	if k.Target != "" {
		f := set.fit(labelFace, k.Target, dialWidth, 13, 10, 1)
		set.draw(im, 1, labelFace, f.px, x, 10, f.lines[0], target)
	}
	number, unit := k.Value, ""
	if head, tail, ok := strings.Cut(k.Value, " "); ok && (tail == "dB" || tail == "%") {
		number, unit = head, tail
	}
	unitWidth := 0
	if unit != "" {
		unitWidth = set.width(labelFace, 16, unit) + 6
	}
	value := set.fit(valueFace, strings.ToUpper(number), dialWidth-unitWidth, 44, 20, 1)
	set.draw(im, 1, valueFace, value.px, x-1, 28, value.lines[0], ink)
	if unit != "" {
		end := x + set.width(valueFace, value.px, value.lines[0]) + 4
		set.draw(im, 1, labelFace, 16, end, 28+value.px-20, unit, offValue)
	}
	if k.Inactive {
		return // No position or level while the control has no effect.
	}
	if k.PositionKnown {
		drawPosition(im, x, dialWidth, k)
	}
	if k.Meter {
		drawMeter(im, x0, k)
	} else if k.Name != "" {
		f := set.fit(labelFace, k.Name, dialWidth, 12, 10, 1)
		set.draw(im, 1, labelFace, f.px, x, 84, f.lines[0], offValue)
	}
}

// drawPosition shows where the value sits in its range, with an optional
// neutral mark (0 dB for gain).
func drawPosition(im *image.RGBA, x, width int, k knobPresentation) {
	fill(im, x, trackTop, x+width, trackTop+4, trackColor)
	fill(im, x, trackTop, x+int(k.Position*float64(width)+0.5), trackTop+4, keyArea(k.Icon))
	if k.ZeroMark > 0 {
		zx := x + int(k.ZeroMark*float64(width)+0.5)
		fill(im, zx-1, trackTop-3, zx+1, trackTop+7, lightValue)
	}
}

// meterZone is the flat zone colour for a level: green below -12 dB, amber
// to -3 dB, red above.
func meterZone(db float64) color.RGBA {
	switch {
	case db < -12:
		return meterGreenColor
	case db < -3:
		return meterAmberColor
	}
	return meterRedColor
}

// dim keeps an unlit zone faintly visible.
func dim(c color.RGBA) color.RGBA {
	return blend(keyColor, c, 0.22)
}

// drawMeter fills the panel's bottom edge across -60..0 dBFS in flat zones,
// with a held peak tick. Unknown or expired readings say so instead of
// drawing silence.
func drawMeter(im *image.RGBA, x0 int, k knobPresentation) {
	if !k.LevelKnown {
		set := typefaces()
		set.draw(im, 1, labelFace, 12, x0+panelInset, 86, "Level N/A", unknownColor)
		return
	}
	for px := 0; px < panelWidth; px++ {
		db := -60 + float64(px)/float64(panelWidth-1)*60
		c := meterZone(db)
		if db >= k.LevelDB {
			c = dim(c)
		}
		fill(im, x0+px, meterTop, x0+px+1, meterTop+meterRows, c)
	}
	if k.PeakKnown && k.PeakDB > -60 {
		px := x0 + int((min(0, k.PeakDB)+60)/60*float64(panelWidth-1))
		ink := lightValue
		if k.PeakDB >= -3 {
			ink = meterRedColor
		}
		fill(im, max(x0, px-1), meterTop-4, min(x0+panelWidth, px+2), meterTop+meterRows, ink)
	}
}

// drawArtworkKnob puts artwork on the left, the target, name or status and
// value beside it, and the progress bar along the bottom. A title or name
// too long to fit scrolls; it reports whether one does.
func drawArtworkKnob(im *image.RGBA, x0 int, k knobPresentation, phase time.Duration) bool {
	set := typefaces()
	scrolls := false
	x, width := x0+110, panelWidth-110-8
	if artworkKind(k.Icon) == programArt {
		// A program icon stays icon-sized beside the text.
		drawThumb(im, x0+panelInset, 22, 56, k.Artwork, k.Icon)
		x, width = x0+80, panelWidth-80-8
	} else {
		drawThumb(im, x0, 0, 100, k.Artwork, k.Icon)
	}
	line := func(s string, maxPx, minPx, y int, ink color.RGBA) {
		if f, ok := set.fitWords(labelFace, s, width, maxPx, minPx, 1); ok {
			set.draw(im, 1, labelFace, f.px, x, y, f.lines[0], ink)
			return
		}
		set.drawScrolling(im, 1, labelFace, minPx, x, y, width, s, ink, phase)
		scrolls = true
	}
	if k.Target != "" {
		line(k.Target, 14, 12, 10, keyArea(k.Icon))
	}
	note, ink := k.Name, offValue
	if k.Status != "" {
		note, ink = k.Status, attentionColor
	}
	if note != "" {
		line(note, 12, 11, 30, ink)
	}
	value := k.Value
	if set.width(valueFace, 20, value) > width {
		value = strings.ReplaceAll(value, " / ", "/")
	}
	f := set.fit(valueFace, value, width, 20, 14, 1)
	set.draw(im, 1, valueFace, f.px, x, 56, f.lines[0], lightValue)
	if k.PositionKnown {
		fill(im, x0, 94, x0+panelWidth, 100, trackColor)
		fill(im, x0, 94, x0+int(k.Position*panelWidth+0.5), 100, keyArea(k.Icon))
	}
	return scrolls
}

// drawThumb draws artwork, or the icon when there is none, smoothly scaled
// into a size-pixel square at x, y.
func drawThumb(im *image.RGBA, x, y, size int, artwork, icon string) {
	target := image.Rect(x, y, x+size, y+size)
	if source, ok := decodeArtwork(artwork); ok {
		xdraw.CatmullRom.Scale(im, target, source, source.Bounds(), draw.Over, nil)
		return
	}
	drawIconBox(im, 1, icon, lightValue, x, y, size)
}

// drawTimers lays out one countdown across a dial panel, or two in rows.
func drawTimers(im *image.RGBA, x int, timers [2]TimerTile) {
	set := typefaces()
	if timers[1] == (TimerTile{}) {
		t := timers[0]
		drawThumb(im, x+8, 18, 64, t.Artwork, t.Icon)
		width := panelWidth - 84 - 8
		f := set.fit(valueFace, t.Time, width, 36, 20, 1)
		set.draw(im, 1, valueFace, f.px, x+84, 20, f.lines[0], lightValue)
		label := set.fit(labelFace, t.Label, width, 12, 10, 1)
		set.draw(im, 1, labelFace, label.px, x+84, 66, label.lines[0], offValue)
		return
	}
	for row, t := range timers {
		y := 6 + row*48
		width := panelWidth - 56 - 8
		drawThumb(im, x+8, y, 40, t.Artwork, t.Icon)
		f := set.fit(valueFace, t.Time, width, 24, 16, 1)
		set.draw(im, 1, valueFace, f.px, x+56, y+2, f.lines[0], lightValue)
		label := set.fit(labelFace, t.Label, width, 11, 9, 1)
		set.draw(im, 1, labelFace, label.px, x+56, y+28, label.lines[0], offValue)
	}
}
