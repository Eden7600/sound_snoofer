package hue

import (
	"math"
	"sort"
	"strings"
)

// Scene sort orders (openspec/changes/hue-scene-sort). Empty means name.
const (
	sortName       = "name"
	sortHue        = "hue"
	sortBrightness = "brightness"
)

var sceneSorts = []string{sortName, sortHue, sortBrightness}

var sceneSortLabels = map[string]string{sortName: "Name", sortHue: "Hue", sortBrightness: "Brightness"}

// whiteSaturation is the HSV saturation below which a scene's main colour
// counts as white and sorts after every colour.
const whiteSaturation = 0.25

// sceneKey is a scene's position in the hue and brightness orders. Lower
// rank sorts first; within a rank, value orders ascending.
type sceneKey struct {
	rank  int
	value float64
}

// hueKey ranks colours by their main wedge's hue angle, then whites, then
// scenes without colour data.
func hueKey(scene resource) sceneKey {
	wedges := mergeSwatches(sceneSwatches(scene))
	if len(wedges) == 0 {
		return sceneKey{rank: 2}
	}
	angle, saturation := hueSaturation(wedges[0])
	if saturation < whiteSaturation {
		return sceneKey{rank: 1}
	}
	return sceneKey{value: math.Round(angle)} // Whole degrees, so near-equal colours sort by name.
}

// brightnessKey orders brightest first, then scenes without brightness data.
func brightnessKey(scene resource) sceneKey {
	b, ok := sceneBrightness(scene)
	if !ok {
		return sceneKey{rank: 1}
	}
	return sceneKey{value: -math.Round(b * 100)} // Whole percent, as near-equal scenes look alike.
}

// hueSaturation is a colour's HSV hue angle, 0–360°, and saturation, 0–1.
func hueSaturation(s swatch) (angle, saturation float64) {
	high := math.Max(s.R, math.Max(s.G, s.B))
	low := math.Min(s.R, math.Min(s.G, s.B))
	if high <= 0 || high == low {
		return 0, 0
	}
	spread := high - low
	switch high {
	case s.R:
		angle = math.Mod((s.G-s.B)/spread, 6)
	case s.G:
		angle = (s.B-s.R)/spread + 2
	default:
		angle = (s.R-s.G)/spread + 4
	}
	angle *= 60
	if angle < 0 {
		angle += 360
	}
	return angle, spread / high
}

// sortScenes orders scenes by the chosen sort; name, then scene ID, breaks ties.
func (m model) sortScenes(scenes []sceneInfo, mode string) {
	keys := make(map[string]sceneKey, len(scenes))
	for _, scene := range scenes {
		switch mode {
		case sortHue:
			keys[scene.SceneID] = hueKey(m.resources[scene.SceneID])
		case sortBrightness:
			keys[scene.SceneID] = brightnessKey(m.resources[scene.SceneID])
		}
	}
	sort.SliceStable(scenes, func(i, j int) bool {
		a, b := keys[scenes[i].SceneID], keys[scenes[j].SceneID]
		if a.rank != b.rank {
			return a.rank < b.rank
		}
		if a.value != b.value {
			return a.value < b.value
		}
		x, y := strings.ToLower(scenes[i].ShortLabel), strings.ToLower(scenes[j].ShortLabel)
		if x != y {
			return x < y
		}
		return scenes[i].SceneID < scenes[j].SceneID
	})
}
