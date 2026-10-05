package streamdeck

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
)

// drawIcon uses code-drawn shapes so symbols do not depend on installed fonts.
func drawIcon(im *image.RGBA, icon string, ink color.RGBA) bool {
	scale := im.Bounds().Dx() / 112
	rect := func(x, y, w, h int) {
		draw.Draw(im, image.Rect(x*scale, (y+14)*scale, (x+w)*scale, (y+h+14)*scale), &image.Uniform{ink}, image.Point{}, draw.Src)
	}
	dot := func(x, y, r int) {
		x *= scale
		y = (y + 14) * scale
		r *= scale
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if dx*dx+dy*dy <= r*r {
					im.SetRGBA(x+dx, y+dy, ink)
				}
			}
		}
	}
	line := func(x1, y1, x2, y2 int) {
		steps := max(abs(x2-x1), abs(y2-y1))
		for n := 0; n <= steps; n++ {
			x, y := x1, y1
			if steps > 0 {
				x += n * (x2 - x1) / steps
				y += n * (y2 - y1) / steps
			}
			dot(x, y, 2)
		}
	}
	triangle := func(x, y, size, dir int) {
		for dx := 0; dx < size; dx++ {
			half := (size - dx) / 2
			rect(x+dir*dx, y-half, 2, half*2+1)
		}
	}
	muted := strings.HasSuffix(icon, "-muted")
	icon = strings.TrimSuffix(strings.TrimSuffix(icon, "-muted"), "-off")
	switch icon {
	case "record-toggle", "record-start":
		for a := 0; a < 360; a++ {
			r := float64(a) * math.Pi / 180
			dot(56+int(20*math.Cos(r)), 38+int(20*math.Sin(r)), 2)
		}
	case "soundboard-play":
		triangle(40, 38, 36, 1)
	case "record-stop", "soundboard-stop":
		rect(36, 18, 40, 40)
	case "media-prev":
		rect(25, 20, 5, 36)
		triangle(51, 38, 23, -1)
		triangle(79, 38, 23, -1)
	case "media-next":
		triangle(30, 38, 23, 1)
		triangle(58, 38, 23, 1)
		rect(84, 20, 5, 36)
	case "media-play":
		triangle(27, 38, 30, 1)
		rect(68, 22, 7, 32)
		rect(82, 22, 7, 32)
	case "mic-mute", "record-mic", "vr-mic":
		for a := 180; a <= 360; a++ {
			r := float64(a) * math.Pi / 180
			dot(56+int(9*math.Cos(r)), 25+int(9*math.Sin(r)), 2)
		}
		for a := 0; a <= 180; a++ {
			r := float64(a) * math.Pi / 180
			dot(56+int(9*math.Cos(r)), 43+int(9*math.Sin(r)), 2)
		}
		line(47, 25, 47, 43)
		line(65, 25, 65, 43)
		line(39, 38, 39, 48)
		line(39, 48, 46, 57)
		line(46, 57, 66, 57)
		line(66, 57, 73, 48)
		line(73, 48, 73, 38)
		line(56, 57, 56, 66)
		line(45, 66, 67, 66)
	case "speaker-mute", "a1-mute", "a2-mute", "vr-playback":
		line(29, 29, 42, 29)
		line(29, 29, 29, 52)
		line(29, 52, 42, 52)
		line(42, 29, 60, 12)
		line(60, 12, 60, 69)
		line(60, 69, 42, 52)
		line(71, 25, 78, 33)
		line(78, 33, 78, 46)
		line(78, 46, 71, 54)
	case "record-computer":
		line(29, 17, 83, 17)
		line(29, 17, 29, 51)
		line(83, 17, 83, 51)
		line(29, 51, 83, 51)
		line(56, 51, 56, 62)
		line(43, 62, 69, 62)
	case "monitor":
		for a := 180; a <= 360; a++ {
			r := float64(a) * math.Pi / 180
			dot(56+int(26*math.Cos(r)), 40+int(26*math.Sin(r)), 2)
		}
		rect(28, 37, 10, 24)
		rect(75, 37, 10, 24)
	case "mic-stack":
		for a := -45; a <= 225; a++ {
			r := float64(a) * math.Pi / 180
			dot(56+int(23*math.Cos(r)), 40+int(23*math.Sin(r)), 2)
		}
		line(56, 12, 56, 39)
	case "mode-direct":
		dot(24, 39, 4)
		line(28, 39, 86, 39)
		line(76, 29, 86, 39)
		line(76, 49, 86, 39)
	case "mode-element", "tap-pre", "tap-post":
		// Signal flows left-to-right through FX; the capture tap branches before/after.
		line(19, 32, 40, 32)
		line(72, 32, 91, 32)
		line(84, 25, 91, 32)
		line(84, 39, 91, 32)
		line(40, 16, 72, 16)
		line(40, 16, 40, 48)
		line(72, 16, 72, 48)
		line(40, 48, 72, 48)
		text(im, 45*scale, 27*scale+14*scale, scale, "FX", ink)
		if icon == "mode-element" {
			line(46, 60, 66, 60)
			dot(50, 60, 3)
		} else {
			tap := 29
			if icon == "tap-post" {
				tap = 81
			}
			dot(tap, 32, 3)
			line(tap, 32, tap, 59)
			dot(tap, 63, 5)
		}
	case "open-controls", "defaults":
		for n, x := range []int{33, 56, 79} {
			line(x, 16, x, 63)
			y := []int{27, 48, 35}[n]
			for a := 0; a < 360; a++ {
				r := float64(a) * math.Pi / 180
				dot(x+int(6*math.Cos(r)), y+int(6*math.Sin(r)), 1)
			}
		}
	case "engine-restart":
		for a := 30; a < 330; a++ {
			r := float64(a) * math.Pi / 180
			dot(56+int(23*math.Cos(r)), 39+int(23*math.Sin(r)), 2)
		}
		line(76, 27, 84, 29)
		line(84, 29, 84, 18)
	default:
		return false
	}
	if muted {
		line(25, 14, 87, 67)
	}
	return true
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
