package streamdeck

import (
	"bytes"

	"image"
	"image/color"
	"image/jpeg"
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
	// Supersample geometry; text remains aligned to the native key pixel grid.
	const scale = 4
	im := image.NewRGBA(image.Rect(0, 0, w*scale, h*scale))
	background := color.RGBA{14, 20, 27, 255}
	foreground := color.RGBA{227, 237, 243, 255}
	accent := color.RGBA{135, 151, 163, 255}
	label, value := "", ""
	if len(lines) > 0 {
		label = lines[0]
	}
	if len(lines) > 1 {
		value = strings.ToUpper(lines[1])
	}
	if on || value == "LIVE" || value == "AUDIBLE" || value == "ELEMENT" || strings.Contains(value, "VST") {
		accent = color.RGBA{58, 198, 225, 255}
	}
	if strings.HasSuffix(icon, "-muted") || icon == "record-stop" {
		accent = color.RGBA{255, 105, 120, 255}
	}
	if value == "PENDING" || value == "UNAVAIL" || value == "ERROR" || fallback {
		accent = color.RGBA{238, 183, 76, 255}
	}
	if value == "UNAVAILABLE" {
		foreground = color.RGBA{80, 90, 100, 255}
		accent = foreground
	}
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
	rect := func(x, y, width, height int, c color.RGBA) {
		for yy := y * scale; yy < (y+height)*scale; yy++ {
			for xx := x * scale; xx < (x+width)*scale; xx++ {
				im.SetRGBA(xx, yy, c)
			}
		}
	}
	rect(0, 0, w, h, background)
	if label != "" {
		rect(3, 3, w-6, 1, accent)
		rect(3, h-4, w-6, 1, accent)
		rect(3, 3, 1, h-6, accent)
		rect(w-4, 3, 1, h-6, accent)
	}
	ink := foreground
	if accent.G > 180 && accent.B > 180 {
		ink = accent
	}
	iconDrawn := drawIcon(im, icon, ink)
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
	Label, Value, Icon string
	Fallback           bool
}
type knobPresentation struct{ Target, Value, Name, Status string }
type presentation struct {
	Keys  [Keys]keyPresentation
	Knobs [Encoders]knobPresentation
}

func renderPresentation(view presentation) ([][]byte, []byte) {
	tiles := make([][]byte, Keys)
	for n, key := range view.Keys {
		var lines []string
		if key.Label != "" {
			lines = []string{key.Label, key.Value}
		}
		tiles[n] = render(lines, 112, 112, key.Value == "On" || key.Value == "Recording", key.Icon, key.Fallback)
	}
	touchImage := image.NewRGBA(image.Rect(0, 0, 1200, 100))
	for n, knob := range view.Knobs {
		text(touchImage, n*200+8, 12, 2, knob.Target, color.RGBA{30, 200, 220, 255})
		text(touchImage, n*200+100, 12, 2, knob.Status, color.RGBA{255, 170, 60, 255})
		text(touchImage, n*200+8, 40, 2, knob.Value, color.RGBA{240, 240, 240, 255})
		text(touchImage, n*200+8, 72, 2, knob.Name, color.RGBA{150, 160, 170, 255})
	}
	rotated := image.NewRGBA(image.Rect(0, 0, 100, 1200))
	for y := 0; y < 100; y++ {
		for x := 0; x < 1200; x++ {
			rotated.SetRGBA(y, 1199-x, touchImage.RGBAAt(x, y))
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, rotated, &jpeg.Options{Quality: 80})
	return tiles, buf.Bytes()
}
