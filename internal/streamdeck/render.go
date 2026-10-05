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
	'A': {126, 9, 9, 9, 126}, 'B': {127, 73, 73, 73, 54}, 'C': {62, 65, 65, 65, 34}, 'D': {127, 65, 65, 34, 28}, 'E': {127, 73, 73, 73, 65}, 'F': {127, 9, 9, 9, 1}, 'G': {62, 65, 73, 73, 122}, 'H': {127, 8, 8, 8, 127}, 'I': {65, 65, 127, 65, 65}, 'J': {32, 64, 65, 63, 1}, 'K': {127, 8, 20, 34, 65}, 'L': {127, 64, 64, 64, 64}, 'M': {127, 2, 12, 2, 127}, 'N': {127, 4, 8, 16, 127}, 'O': {62, 65, 65, 65, 62}, 'P': {127, 9, 9, 9, 6}, 'Q': {62, 65, 81, 33, 94}, 'R': {127, 9, 25, 41, 70}, 'S': {38, 73, 73, 73, 50}, 'T': {1, 1, 127, 1, 1}, 'U': {63, 64, 64, 64, 63}, 'V': {31, 32, 64, 32, 31}, 'W': {63, 64, 56, 64, 63}, 'X': {99, 20, 8, 20, 99}, 'Y': {7, 8, 112, 8, 7}, 'Z': {97, 81, 73, 69, 67},
	'0': {62, 81, 73, 69, 62}, '1': {0, 66, 127, 64, 0}, '2': {66, 97, 81, 73, 70}, '3': {33, 65, 69, 75, 49}, '4': {24, 20, 18, 127, 16}, '5': {39, 69, 69, 69, 57}, '6': {60, 74, 73, 73, 48}, '7': {1, 113, 9, 5, 3}, '8': {54, 73, 73, 73, 54}, '9': {6, 73, 73, 41, 30}, '-': {8, 8, 8, 8, 8}, '.': {0, 96, 96, 0, 0}, '/': {32, 16, 8, 4, 2}, '?': {2, 1, 81, 9, 6}, '+': {8, 8, 62, 8, 8}, '*': {20, 8, 62, 8, 20}}

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
		text(panel, n*200+8, 12, 2, knob.Target, activeColor)
		text(panel, n*200+100, 12, 2, knob.Status, attentionColor)
		text(panel, n*200+8, 40, 2, knob.Value, textColor)
		if knob.Meter {
			drawMeter(panel, n*200+8, knob)
		} else {
			text(panel, n*200+8, 72, 2, knob.Name, neutralColor)
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

func drawMeter(im *image.RGBA, x int, k knobPresentation) {
	if !k.LevelKnown {
		text(im, x, 72, 1, "LEVEL N/A", neutralColor)
		return
	}
	for segment := 0; segment < 24; segment++ {
		threshold := -60 + float64(segment)*2.5
		c := meterQuietColor
		if k.LevelDB > threshold {
			c = meterGreenColor
			if threshold >= -12 {
				c = meterAmberColor
			}
			if threshold >= -3 {
				c = meterRedColor
			}
		}
		for yy := 68; yy < 81; yy++ {
			for xx := x + segment*7; xx < x+segment*7+5; xx++ {
				im.SetRGBA(xx, yy, c)
			}
		}
	}
	scale := neutralColor
	text(im, x, 87, 1, "-60", scale)
	text(im, x+75, 87, 1, "-30", scale)
	text(im, x+114, 87, 1, "DBFS", scale)
	text(im, x+162, 87, 1, "0", scale)
}

// drawArtwork accepts only the bounded thumbnail contract, never source files.
func drawArtwork(im *image.RGBA, artwork string) bool {
	if artwork == "" || len(artwork) > 32768 {
		return false
	}
	data, err := base64.StdEncoding.DecodeString(artwork)
	if err != nil {
		return false
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Width != cfg.Height || cfg.Width > 64 {
		return false
	}
	source, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return false
	}
	scale := im.Bounds().Dx() / 112
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			pixel := source.At(x*cfg.Width/64, y*cfg.Height/64)
			draw.Draw(im, image.Rect((24+x)*scale, (20+y)*scale, (25+x)*scale, (21+y)*scale), &image.Uniform{pixel}, image.Point{}, draw.Over)
		}
	}
	return true
}
