package hue

import (
	"math"
	"sort"
)

// Light groups and saturation for the Decor, Mixed and Saturation dials
// (openspec/changes/hue-light-groups).

// gamut is the triangle of colours a light can show, in CIE xy.
type gamut struct {
	Red   xyPoint `json:"red"`
	Green xyPoint `json:"green"`
	Blue  xyPoint `json:"blue"`
}

var (
	gamutC     = gamut{Red: xyPoint{0.6915, 0.3083}, Green: xyPoint{0.17, 0.7}, Blue: xyPoint{0.1532, 0.0475}}
	whitePoint = xyPoint{0.3127, 0.3290} // D65.
)

// nearWhite is the distance from white within which a colour has no hue.
const nearWhite = 0.005

// Light functions as the bridge reports them.
const (
	decorative = "decorative"
	mixed      = "mixed"
)

// roomLight is one light of a room, as the light-group dials see it.
type roomLight struct {
	ID, Function string
	OnKnown, On  bool
	BrightKnown  bool
	Brightness   float64
	Colour       bool      // Colour-capable; white-only lamps are not.
	ColourMode   bool      // Showing a colour rather than a white temperature.
	XY           *xyPoint  // Current colour, when known.
	Points       []xyPoint // Gradient points shown, if any.
	Gamut        gamut
}

// roomLights lists a room's lights sorted by ID.
func (m model) roomLights(groupID string) []roomLight {
	g, ok := m.resources[groupID]
	if !ok || (g.Type != "room" && g.Type != "zone") {
		return nil
	}
	var out []roomLight
	for _, r := range m.lights(g) {
		out = append(out, lightView(r))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// lightView reads one light resource as the light-group dials see it.
func lightView(r resource) roomLight {
	light := roomLight{ID: r.ID, Gamut: gamutC}
	if r.Metadata != nil {
		light.Function = r.Metadata.Function
	}
	if r.On != nil {
		light.OnKnown, light.On = true, r.On.On
	}
	if r.Dimming != nil {
		light.BrightKnown, light.Brightness = true, r.Dimming.Brightness
	}
	if r.Color != nil {
		light.Colour = true
		light.XY = r.Color.XY
		if r.Color.Gamut != nil {
			light.Gamut = *r.Color.Gamut
		}
		white := r.ColorTemperature != nil && r.ColorTemperature.MirekValid != nil && *r.ColorTemperature.MirekValid
		light.ColourMode = !white
	}
	if r.Gradient != nil {
		for _, p := range r.Gradient.Points {
			light.Points = append(light.Points, p.Color.XY)
		}
	}
	return light
}

// withFunction returns the lights with the given function.
func withFunction(lights []roomLight, function string) []roomLight {
	var out []roomLight
	for _, l := range lights {
		if l.Function == function {
			out = append(out, l)
		}
	}
	return out
}

// colourLights returns the colour-capable lights.
func colourLights(lights []roomLight) []roomLight {
	var out []roomLight
	for _, l := range lights {
		if l.Colour {
			out = append(out, l)
		}
	}
	return out
}

// groupState summarizes a group's lights: whether any is lit, and the
// average brightness of those that are. Known is false when any light's
// on state is unreported.
type groupState struct {
	Known, AnyOn bool
	Brightness   float64 // Average of lit lights with a reported brightness.
	BrightKnown  bool
}

func summarize(lights []roomLight) groupState {
	state := groupState{Known: len(lights) > 0}
	total, counted := 0.0, 0
	for _, l := range lights {
		if !l.OnKnown {
			state.Known = false
			continue
		}
		if !l.On {
			continue
		}
		state.AnyOn = true
		if l.BrightKnown {
			total += l.Brightness
			counted++
		}
	}
	if counted > 0 {
		state.Brightness, state.BrightKnown = total/float64(counted), true
	}
	return state
}

// edgeDistance is the distance from white to the gamut's edge along unit
// direction (dx, dy), or 0 when the ray misses the triangle.
func edgeDistance(g gamut, dx, dy float64) float64 {
	best := math.Inf(1)
	for _, edge := range [][2]xyPoint{{g.Red, g.Green}, {g.Green, g.Blue}, {g.Blue, g.Red}} {
		ax, ay := edge[0].X, edge[0].Y
		ex, ey := edge[1].X-ax, edge[1].Y-ay
		// Solve white + t*d = a + u*e.
		den := dx*ey - dy*ex
		if math.Abs(den) < 1e-12 {
			continue
		}
		wx, wy := ax-whitePoint.X, ay-whitePoint.Y
		t := (wx*ey - wy*ex) / den
		u := (wx*dy - wy*dx) / den
		if t > 0 && u >= 0 && u <= 1 && t < best {
			best = t
		}
	}
	if math.IsInf(best, 1) {
		return 0
	}
	return best
}

// saturation is how far a colour lies from white toward the gamut's edge, 0
// to 1. ok is false for colours too near white to have a hue.
func saturation(xy xyPoint, g gamut) (value float64, ok bool) {
	dx, dy := xy.X-whitePoint.X, xy.Y-whitePoint.Y
	distance := math.Hypot(dx, dy)
	if distance < nearWhite {
		return 0, false
	}
	edge := edgeDistance(g, dx/distance, dy/distance)
	if edge == 0 {
		return 0, false
	}
	return math.Min(1, distance/edge), true
}

// withSaturation moves a colour along its own line from white to target (0
// to 1) of the way to the gamut's edge, keeping its hue. Colours too near
// white are returned unchanged.
func withSaturation(xy xyPoint, g gamut, target float64) xyPoint {
	dx, dy := xy.X-whitePoint.X, xy.Y-whitePoint.Y
	distance := math.Hypot(dx, dy)
	if distance < nearWhite {
		return xy
	}
	ux, uy := dx/distance, dy/distance
	edge := edgeDistance(g, ux, uy)
	if edge == 0 {
		return xy
	}
	// Stay a hair inside the edge so floating error never leaves the gamut.
	reach := edge * math.Max(0, math.Min(1, target)) * 0.999
	return xyPoint{X: whitePoint.X + ux*reach, Y: whitePoint.Y + uy*reach}
}

// lightColours lists the colours a colour-mode light shows: its gradient
// points, or its single colour.
func lightColours(l roomLight) []xyPoint {
	if len(l.Points) > 0 {
		return l.Points
	}
	if l.XY != nil {
		return []xyPoint{*l.XY}
	}
	return nil
}

// averageSaturation is the mean saturation of the colour-mode lights'
// colours, skipping colours too near white. ok is false when there are none.
func averageSaturation(lights []roomLight) (value float64, ok bool) {
	total, counted := 0.0, 0
	for _, l := range lights {
		if !l.ColourMode {
			continue
		}
		for _, xy := range lightColours(l) {
			if s, has := saturation(xy, l.Gamut); has {
				total += s
				counted++
			}
		}
	}
	if counted == 0 {
		return 0, false
	}
	return total / float64(counted), true
}
