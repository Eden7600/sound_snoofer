package hue

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"sort"
	"strings"
)

const (
	artworkSize     = 64 // Thumbnail edge in pixels (the control artwork limit).
	artworkScale    = 4  // Supersampling factor for smooth wedge edges.
	maxWedges       = 5
	mergeDistance   = 40.0 // RGB distance below which colors share a wedge.
	minArtworkLevel = 0.55 // Dimmest wedge brightness, so dim scenes stay visible.
)

// swatch is one scene color with the share of lights that use it.
type swatch struct {
	R, G, B float64 // sRGB, 0-255.
	Weight  float64
}

// sceneSwatches extracts the colors a scene sets on the lights it turns on.
// Colorless actions are ignored; without any action colors the palette is used.
func sceneSwatches(scene resource) []swatch {
	var out []swatch
	actions := append([]sceneAction(nil), scene.Actions...)
	sort.SliceStable(actions, func(i, j int) bool { return actions[i].Target.RID < actions[j].Target.RID })
	for _, a := range actions {
		if a.Action.On != nil && !a.Action.On.On {
			continue
		}
		level := artworkLevel(a.Action.Dimming)
		switch {
		case a.Action.Gradient != nil && len(a.Action.Gradient.Points) > 0:
			share := 1 / float64(len(a.Action.Gradient.Points))
			for _, point := range a.Action.Gradient.Points {
				out = append(out, xySwatch(point.Color.XY, level, share))
			}
		case a.Action.Color != nil:
			out = append(out, xySwatch(a.Action.Color.XY, level, 1))
		case a.Action.ColorTemperature != nil && a.Action.ColorTemperature.Mirek != nil:
			out = append(out, mirekSwatch(*a.Action.ColorTemperature.Mirek, level, 1))
		}
	}
	if len(out) > 0 || scene.Palette == nil {
		return out
	}
	for _, entry := range scene.Palette.Color {
		out = append(out, xySwatch(entry.Color.XY, artworkLevel(entry.Dimming), 1))
	}
	for _, entry := range scene.Palette.ColorTemperature {
		if entry.ColorTemperature.Mirek != nil {
			out = append(out, mirekSwatch(*entry.ColorTemperature.Mirek, artworkLevel(entry.Dimming), 1))
		}
	}
	return out
}

func artworkLevel(d *dimming) float64 {
	if d == nil {
		return 1
	}
	return minArtworkLevel + (1-minArtworkLevel)*math.Min(100, math.Max(0, d.Brightness))/100
}

// xySwatch converts CIE xy to sRGB with the Hue wide-gamut matrix, normalized
// to full brightness before applying level.
func xySwatch(point xyPoint, level, weight float64) swatch {
	if point.Y <= 0 {
		return swatch{R: 255 * level, G: 255 * level, B: 255 * level, Weight: weight}
	}
	z := 1 - point.X - point.Y
	x, y := point.X/point.Y, 1.0
	zz := z / point.Y
	r := x*1.656492 - y*0.354851 - zz*0.255038
	g := -x*0.707196 + y*1.655397 + zz*0.036152
	b := x*0.051713 - y*0.121364 + zz*1.011530
	r, g, b = math.Max(0, r), math.Max(0, g), math.Max(0, b)
	peak := math.Max(r, math.Max(g, b))
	if peak <= 0 {
		return swatch{Weight: weight}
	}
	return swatch{R: 255 * level * gamma(r/peak), G: 255 * level * gamma(g/peak), B: 255 * level * gamma(b/peak), Weight: weight}
}

func gamma(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

// mirekSwatch approximates a white temperature's color (Tanner Helland's fit).
func mirekSwatch(mirek int, level, weight float64) swatch {
	if mirek <= 0 {
		return swatch{R: 255 * level, G: 255 * level, B: 255 * level, Weight: weight}
	}
	t := 1e6 / float64(mirek) / 100
	clamp := func(v float64) float64 { return math.Min(255, math.Max(0, v)) }
	var r, g, b float64
	if t <= 66 {
		r = 255
		g = 99.4708025861*math.Log(t) - 161.1195681661
	} else {
		r = 329.698727446 * math.Pow(t-60, -0.1332047592)
		g = 288.1221695283 * math.Pow(t-60, -0.0755148492)
	}
	switch {
	case t >= 66:
		b = 255
	case t <= 19:
		b = 0
	default:
		b = 138.5177312231*math.Log(t-10) - 305.0447927307
	}
	return swatch{R: clamp(r) * level, G: clamp(g) * level, B: clamp(b) * level, Weight: weight}
}

// mergeSwatches groups near-identical colors and keeps the largest groups.
func mergeSwatches(in []swatch) []swatch {
	var groups []swatch
	for _, s := range in {
		merged := false
		for n := range groups {
			g := &groups[n]
			if math.Hypot(math.Hypot(g.R-s.R, g.G-s.G), g.B-s.B) < mergeDistance {
				total := g.Weight + s.Weight
				g.R = (g.R*g.Weight + s.R*s.Weight) / total
				g.G = (g.G*g.Weight + s.G*s.Weight) / total
				g.B = (g.B*g.Weight + s.B*s.Weight) / total
				g.Weight = total
				merged = true
				break
			}
		}
		if !merged {
			groups = append(groups, s)
		}
	}
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].Weight > groups[j].Weight })
	if len(groups) > maxWedges {
		groups = groups[:maxWedges]
	}
	return groups
}

// artworkKey identifies rendered output, so unchanged scenes reuse their image.
func artworkKey(wedges []swatch) string {
	var b strings.Builder
	for _, w := range wedges {
		fmt.Fprintf(&b, "%.0f,%.0f,%.0f,%.3f;", w.R, w.G, w.B, w.Weight)
	}
	return b.String()
}

// renderArtwork draws wedges clockwise from the top, sized by weight, inside a
// transparent square, and returns a base64 PNG.
func renderArtwork(wedges []swatch) (string, error) {
	total := 0.0
	for _, w := range wedges {
		total += w.Weight
	}
	if total <= 0 {
		return "", nil
	}
	large := artworkSize * artworkScale
	center := float64(large) / 2
	radius := center - float64(artworkScale)
	ends := make([]float64, len(wedges))
	sum := 0.0
	for n, w := range wedges {
		sum += w.Weight
		ends[n] = sum / total
	}
	out := image.NewNRGBA(image.Rect(0, 0, artworkSize, artworkSize))
	for y := 0; y < artworkSize; y++ {
		for x := 0; x < artworkSize; x++ {
			var r, g, b, covered float64
			for sy := 0; sy < artworkScale; sy++ {
				for sx := 0; sx < artworkScale; sx++ {
					dx := float64(x*artworkScale+sx) + 0.5 - center
					dy := float64(y*artworkScale+sy) + 0.5 - center
					if math.Hypot(dx, dy) > radius {
						continue
					}
					// Fraction of a clockwise turn starting at 12 o'clock.
					turn := math.Mod(math.Atan2(dx, -dy)/(2*math.Pi)+1, 1)
					n := sort.SearchFloat64s(ends, turn)
					n = min(n, len(wedges)-1)
					r, g, b, covered = r+wedges[n].R, g+wedges[n].G, b+wedges[n].B, covered+1
				}
			}
			if covered == 0 {
				continue
			}
			samples := float64(artworkScale * artworkScale)
			out.SetNRGBA(x, y, color.NRGBA{R: uint8(r / covered), G: uint8(g / covered), B: uint8(b / covered), A: uint8(255 * covered / samples)})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, out); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

// cachedArtwork is a rendered thumbnail and the color content it shows.
type cachedArtwork struct {
	key, art string
}

// sceneArtwork returns a scene's thumbnail, rendering only when its colors
// changed. It returns "" when the scene has no color data.
func (w *worker) sceneArtwork(sceneID string) string {
	wedges := mergeSwatches(sceneSwatches(w.model.resources[sceneID]))
	if len(wedges) == 0 {
		return ""
	}
	key := artworkKey(wedges)
	if cached, ok := w.artwork[sceneID]; ok && cached.key == key {
		return cached.art
	}
	art, err := renderArtwork(wedges)
	if err != nil {
		art = "" // PNG encoding into memory cannot fail for a valid image; fall back to the icon.
	}
	w.artwork[sceneID] = cachedArtwork{key: key, art: art}
	return art
}
