package streamdeck

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
)

// drawIcon uses code-drawn shapes so symbols do not depend on installed fonts.
func drawIcon(im *image.RGBA, icon string) bool {
	ink := color.RGBA{225, 241, 246, 255}
	rect := func(x, y, w, h int) {
		draw.Draw(im, image.Rect(x, y, x+w, y+h), &image.Uniform{ink}, image.Point{}, draw.Src)
	}
	dot := func(x, y, r int) {
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
	icon = strings.TrimSuffix(icon, "-muted")
	switch icon {
	case "record-toggle", "record-start":
		ink = color.RGBA{255, 86, 100, 255}
		dot(56, 38, 21)
	case "record-stop":
		ink = color.RGBA{255, 110, 120, 255}
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
		dot(56, 25, 9)
		rect(47, 25, 19, 19)
		dot(56, 43, 9)
		line(39, 38, 39, 48)
		line(39, 48, 46, 57)
		line(46, 57, 66, 57)
		line(66, 57, 73, 48)
		line(73, 48, 73, 38)
		line(56, 57, 56, 66)
		line(45, 66, 67, 66)
	case "speaker-mute", "a1-mute", "a2-mute", "vr-playback":
		rect(29, 29, 13, 23)
		for x := 42; x < 60; x++ {
			rect(x, 29-(x-42), 1, 23+2*(x-42))
		}
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
	case "mode", "open-controls", "record-tap", "defaults":
		for n, x := range []int{33, 56, 79} {
			line(x, 16, x, 63)
			y := []int{27, 48, 35}[n]
			rect(x-7, y-4, 15, 9)
		}
	default:
		return false
	}
	if muted {
		ink = color.RGBA{255, 125, 125, 255}
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
