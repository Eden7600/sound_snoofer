package streamdeck

import (
	"bytes"
	"encoding/base64"

	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"strings"
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

func renderArtwork(lines []string, w, h int, on bool, icon string, fallback bool, artwork string) []byte {
	// Supersample geometry; text remains aligned to the native key pixel grid.
	const scale = 4
	im := image.NewRGBA(image.Rect(0, 0, w*scale, h*scale))
	background := backgroundColor
	foreground := textColor

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
	rect := func(x, y, width, height int, c color.RGBA) {
		for yy := y * scale; yy < (y+height)*scale; yy++ {
			for xx := x * scale; xx < (x+width)*scale; xx++ {
				im.SetRGBA(xx, yy, c)
			}
		}
	}
	rect(0, 0, w, h, background)

	ink := foreground
	if accent != neutralColor {
		ink = accent
	}
	iconDrawn := drawArtwork(im, artwork)
	if !iconDrawn {
		iconDrawn = drawIcon(im, icon, ink)
	}
	centered := func(s string, y, size int, c color.RGBA) {
		runes := []rune(strings.ToUpper(s))
		maxChars := (w - 14) / (6 * size)
		if len(runes) > maxChars {
			runes = runes[:maxChars]
		}
		text(im, (w-len(runes)*6*size+size)*scale/2, y*scale, size*scale, string(runes), c)
	}
	if iconDrawn {
		centered(label, 10, 1, foreground)
	} else if label != "" {
		// Custom actions keep readable, wrapped labels instead of clipped text.
		words := strings.Fields(label)
		line := ""
		y := 24
		for _, word := range words {
			if len(line)+len(word)+1 > 8 && line != "" {
				centered(line, y, 2, foreground)
				y += 18
				line = ""
			}
			if line != "" {
				line += " "
			}
			line += word
		}
		if line != "" {
			centered(line, y, 2, foreground)
		}
	}
	if value != "" {
		rect(7, 87, w-14, 19, accent)
		size := 2
		if len([]rune(value)) > 8 {
			size = 1
		}
		y := 90
		if size == 1 {
			y = 93
		}
		centered(value, y, size, background)
	}
	// Average the geometry and rotate 90 degrees counterclockwise for + XL.
	rotated := image.NewRGBA(image.Rect(0, 0, h, w))
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
			rotated.SetRGBA(y, w-1-x, color.RGBA{uint8(r / 16), uint8(g / 16), uint8(b / 16), 255})
		}
	}
	var b bytes.Buffer
	jpeg.Encode(&b, rotated, &jpeg.Options{Quality: 92})
	return b.Bytes()
}

type keyPresentation struct {
	Artwork            string
	Label, Value, Icon string
	Fallback           bool
}
type knobPresentation struct {
	Target, Value, Name, Status string
	Meter, LevelKnown           bool
	LevelDB                     float64
	PeakKnown, PositionKnown    bool
	PeakDB, Position, ZeroMark  float64
	Timers                      [2]TimerTile
	Artwork                     string // Shown left of the dial's text when set.
}
type presentation struct {
	Keys  [Keys]keyPresentation
	Knobs [Encoders]knobPresentation
}

func renderKey(key keyPresentation) []byte {
	var lines []string
	if key.Label != "" {
		lines = []string{key.Label, key.Value}
	}
	return renderArtwork(lines, 112, 112, key.Value == "On" || key.Value == "Recording", key.Icon, key.Fallback, key.Artwork)
}

func renderPresentation(view presentation) ([][]byte, []byte) {
	tiles := make([][]byte, Keys)
	for n, key := range view.Keys {
		tiles[n] = renderKey(key)
	}
	return tiles, renderKnobs(view.Knobs)
}

func renderKnobs(knobs [Encoders]knobPresentation) []byte {
	touchImage := image.NewRGBA(image.Rect(0, 0, 1200, 100))
	draw.Draw(touchImage, touchImage.Bounds(), &image.Uniform{backgroundColor}, image.Point{}, draw.Src)
	for n, knob := range knobs {
		panel := touchImage.SubImage(image.Rect(n*200, 0, (n+1)*200, 100)).(*image.RGBA)
		if n == Encoders-1 {
			// The reserved page dial carries previous/current/next in these fields.
			for row, name := range []string{knob.Target, knob.Value, knob.Name} {
				size, ink := 1, neutralColor
				if row == 1 {
					size, ink = 2, activeColor
				}
				runes := []rune(name)
				limit := 184 / (6 * size)
				if len(runes) > limit {
					runes = append(runes[:limit-3], '.', '.', '.')
				}
				x := n*200 + (200-len(runes)*6*size+size)/2
				text(panel, x, []int{12, 42, 76}[row], size, string(runes), ink)
			}
			continue
		}
		if knob.Timers[0] != (TimerTile{}) {
			drawTimers(panel, n*200, knob.Timers)
			continue
		}
		if knob.Artwork != "" {
			drawArtworkKnob(panel, n*200, knob)
			continue
		}
		x := n*200 + dialInset
		text(panel, x, 8, 2, knob.Target, activeColor)
		text(panel, n*200+100, 8, 2, knob.Status, attentionColor)
		// The value is the dial's headline: large when it fits the panel.
		size := 3
		if len([]rune(knob.Value))*6*size > dialWidth {
			size = 2
		}
		text(panel, x, 28, size, knob.Value, textColor)
		if knob.PositionKnown {
			drawPosition(panel, x, dialWidth, knob)
		}
		if knob.Meter {
			drawMeter(panel, x, knob)
		} else if knob.Name != "" {
			text(panel, x, 72, 2, knob.Name, neutralColor)
		}
	}
	rotated := image.NewRGBA(image.Rect(0, 0, 100, 1200))
	for y := 0; y < 100; y++ {
		for x := 0; x < 1200; x++ {
			rotated.SetRGBA(y, 1199-x, touchImage.RGBAAt(x, y))
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, rotated, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

const (
	dialInset = 8
	dialWidth = 184 // Usable width of a 200px dial panel.
	meterTop  = 64
	meterRows = 12
)

func fill(im *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			im.SetRGBA(x, y, c)
		}
	}
}

// drawPosition shows where the value sits in its range, with an optional
// neutral mark (0 dB for gain).
func drawPosition(im *image.RGBA, x, width int, k knobPresentation) {
	fill(im, x, 54, x+width, 58, meterQuietColor)
	fill(im, x, 54, x+int(k.Position*float64(width)+0.5), 58, activeColor)
	if k.ZeroMark > 0 {
		zx := x + int(k.ZeroMark*float64(width)+0.5)
		fill(im, zx-1, 52, zx+1, 60, textColor)
	}
}

// drawArtworkKnob spans the target across the panel, then puts artwork on the
// left with the value, position track and status or name beside it. The
// value drops the spaces around "/" and then shrinks only when it must.
func drawArtworkKnob(im *image.RGBA, x0 int, k knobPresentation) {
	text(im, x0+dialInset, 6, 2, fit(k.Target, dialWidth/12), activeColor)
	drawThumb(im, x0+dialInset, 28, 56, k.Artwork, "")
	x, width := x0+72, 200-72-dialInset
	value, size := k.Value, 2
	if len([]rune(value))*12 > width {
		value = strings.ReplaceAll(value, " / ", "/")
	}
	if len([]rune(value))*12 > width {
		size = 1
	}
	text(im, x, 34, size, fit(value, width/(6*size)), textColor)
	if k.PositionKnown {
		drawPosition(im, x, width, k)
	}
	if k.Status != "" {
		text(im, x, 68, 1, fit(k.Status, width/6), attentionColor)
	} else if k.Name != "" {
		text(im, x, 68, 1, fit(k.Name, width/6), neutralColor)
	}
}

// meterColor blends green through amber to red across the dBFS scale.
func meterColor(db float64) color.RGBA {
	mix := func(a, b color.RGBA, t float64) color.RGBA {
		t = min(1, max(0, t))
		return color.RGBA{uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t), uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t), uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t), 255}
	}
	if db < -12 {
		return mix(meterGreenColor, meterAmberColor, (db+18)/6)
	}
	return mix(meterAmberColor, meterRedColor, (db+6)/4)
}

// dim keeps the unlit meter faintly visible as a gradient track.
func dim(c color.RGBA) color.RGBA {
	return color.RGBA{uint8((int(c.R)*22 + int(backgroundColor.R)*78) / 100), uint8((int(c.G)*22 + int(backgroundColor.G)*78) / 100), uint8((int(c.B)*22 + int(backgroundColor.B)*78) / 100), 255}
}

// drawMeter renders a continuous gradient bar with a held peak tick. Unknown
// or expired readings say so instead of drawing silence.
func drawMeter(im *image.RGBA, x int, k knobPresentation) {
	if !k.LevelKnown {
		text(im, x, 72, 1, "LEVEL N/A", neutralColor)
		return
	}
	for px := 0; px < dialWidth; px++ {
		db := -60 + float64(px)/float64(dialWidth-1)*60
		c := meterColor(db)
		if db >= k.LevelDB {
			c = dim(c)
		}
		fill(im, x+px, meterTop, x+px+1, meterTop+meterRows, c)
	}
	if k.PeakKnown && k.PeakDB > -60 {
		px := x + int((min(0, k.PeakDB)+60)/60*float64(dialWidth-1))
		ink := textColor
		if k.PeakDB >= -3 {
			ink = meterRedColor
		}
		fill(im, max(x, px-1), meterTop-2, min(x+dialWidth, px+2), meterTop+meterRows+2, ink)
	}
	scale := neutralColor
	for _, mark := range []struct {
		db    float64
		label string
	}{{-60, "-60"}, {-30, "-30"}, {-12, "-12"}, {0, "0"}} {
		px := x + int((mark.db+60)/60*float64(dialWidth-1))
		width := len(mark.label) * 6
		lx := min(px, x+dialWidth-width)
		if mark.db > -60 && mark.db < 0 {
			lx = px - width/2
		}
		text(im, lx, meterTop+meterRows+5, 1, mark.label, scale)
	}
}

// decodeArtwork accepts only the bounded thumbnail contract, never source files.
func decodeArtwork(artwork string) (image.Image, bool) {
	if artwork == "" || len(artwork) > 32768 {
		return nil, false
	}
	data, err := base64.StdEncoding.DecodeString(artwork)
	if err != nil {
		return nil, false
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Width != cfg.Height || cfg.Width > 64 {
		return nil, false
	}
	source, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	return source, true
}

func drawArtwork(im *image.RGBA, artwork string) bool {
	source, ok := decodeArtwork(artwork)
	if !ok {
		return false
	}
	scale := im.Bounds().Dx() / 112
	bounds := source.Bounds()
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			pixel := source.At(bounds.Min.X+x*bounds.Dx()/64, bounds.Min.Y+y*bounds.Dy()/64)
			draw.Draw(im, image.Rect((24+x)*scale, (20+y)*scale, (25+x)*scale, (21+y)*scale), &image.Uniform{pixel}, image.Point{}, draw.Over)
		}
	}
	return true
}

// drawThumb draws artwork, or the icon when there is none, in a size-pixel
// square at x, y.
func drawThumb(im *image.RGBA, x, y, size int, artwork, icon string) {
	source, ok := decodeArtwork(artwork)
	if !ok {
		glyph := image.NewRGBA(image.Rect(0, 0, 112, 112))
		drawIcon(glyph, icon, textColor)
		source = glyph.SubImage(image.Rect(16, 12, 96, 92)) // The icon area of a key.
	}
	bounds := source.Bounds()
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			pixel := source.At(bounds.Min.X+dx*bounds.Dx()/size, bounds.Min.Y+dy*bounds.Dy()/size)
			draw.Draw(im, image.Rect(x+dx, y+dy, x+dx+1, y+dy+1), &image.Uniform{pixel}, image.Point{}, draw.Over)
		}
	}
}

// fit truncates s to at most chars glyphs, marking the cut.
func fit(s string, chars int) string {
	runes := []rune(s)
	if len(runes) <= chars {
		return s
	}
	if chars < 4 {
		return string(runes[:max(0, chars)])
	}
	return string(runes[:chars-3]) + "..."
}

// drawTimers lays out one countdown across a dial panel, or two in rows.
func drawTimers(im *image.RGBA, x int, timers [2]TimerTile) {
	if timers[1] == (TimerTile{}) {
		t := timers[0]
		drawThumb(im, x+8, 18, 64, t.Artwork, t.Icon)
		size := 4
		if len(t.Time)*6*size > 200-84-dialInset {
			size = 3
		}
		text(im, x+84, 26, size, t.Time, textColor)
		text(im, x+84, 66, 1, fit(t.Label, (200-84-dialInset)/6), neutralColor)
		return
	}
	for row, t := range timers {
		y := 6 + row*48
		drawThumb(im, x+8, y, 40, t.Artwork, t.Icon)
		text(im, x+56, y+10, 3, t.Time, textColor)
		text(im, x+56, y+34, 1, fit(t.Label, (200-56-dialInset)/6), neutralColor)
	}
}
