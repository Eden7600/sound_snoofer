package hue

import (
	"math"
	"sort"
	"strings"
)

type reference struct {
	RID   string `json:"rid"`
	RType string `json:"rtype"`
}

type metadata struct {
	Name string `json:"name"`
}

type onState struct {
	On bool `json:"on"`
}

type dimming struct {
	Brightness float64 `json:"brightness"`
}

type mirekSchema struct {
	Minimum int `json:"mirek_minimum"`
	Maximum int `json:"mirek_maximum"`
}

// colorTemperature keeps optional fields as pointers so partial events merge
// without turning missing values into valid zeros.
type colorTemperature struct {
	Mirek  *int         `json:"mirek"`
	Valid  *bool        `json:"mirek_valid"`
	Schema *mirekSchema `json:"mirek_schema"`
}

type sceneStatus struct {
	Active string `json:"active"`
}

// resource is the subset of CLIP v2 resources this plugin reads. Event updates
// contain only changed fields, so every field is optional.
type resource struct {
	ID               string            `json:"id"`
	Type             string            `json:"type"`
	Metadata         *metadata         `json:"metadata,omitempty"`
	Children         []reference       `json:"children,omitempty"`
	Services         []reference       `json:"services,omitempty"`
	Group            *reference        `json:"group,omitempty"`
	On               *onState          `json:"on,omitempty"`
	Dimming          *dimming          `json:"dimming,omitempty"`
	ColorTemperature *colorTemperature `json:"color_temperature,omitempty"`
	Status           *sceneStatus      `json:"status,omitempty"`
}

// merge applies the fields present in an update.
func (r *resource) merge(update resource) {
	if update.Metadata != nil {
		r.Metadata = update.Metadata
	}
	if update.Children != nil {
		r.Children = update.Children
	}
	if update.Services != nil {
		r.Services = update.Services
	}
	if update.Group != nil {
		r.Group = update.Group
	}
	if update.On != nil {
		r.On = update.On
	}
	if update.Dimming != nil {
		r.Dimming = update.Dimming
	}
	if update.Status != nil {
		r.Status = update.Status
	}
	if update.ColorTemperature != nil {
		if r.ColorTemperature == nil {
			r.ColorTemperature = &colorTemperature{}
		}
		next := *r.ColorTemperature
		if update.ColorTemperature.Mirek != nil {
			next.Mirek = update.ColorTemperature.Mirek
		}
		if update.ColorTemperature.Valid != nil {
			next.Valid = update.ColorTemperature.Valid
		}
		if update.ColorTemperature.Schema != nil {
			next.Schema = update.ColorTemperature.Schema
		}
		r.ColorTemperature = &next
	}
}

func (r resource) name() string {
	if r.Metadata == nil {
		return ""
	}
	return r.Metadata.Name
}

// model is the worker-owned view of bridge resources.
type model struct {
	resources map[string]resource
}

func newModel(items []resource) model {
	m := model{resources: make(map[string]resource, len(items))}
	for _, item := range items {
		m.resources[item.ID] = item
	}
	return m
}

// apply merges event-stream changes. Updates for unknown resources are ignored;
// the next full reload restores them.
func (m model) apply(events []event) {
	for _, e := range events {
		for _, item := range e.Data {
			switch e.Type {
			case "add":
				m.resources[item.ID] = item
			case "delete":
				delete(m.resources, item.ID)
			case "update":
				current, ok := m.resources[item.ID]
				if !ok {
					continue
				}
				current.merge(item)
				m.resources[item.ID] = current
			}
		}
	}
}

type groupInfo struct {
	ID, Name, Kind string
}

// groups lists rooms and zones sorted by name.
func (m model) groups() []groupInfo {
	var out []groupInfo
	for _, r := range m.resources {
		if r.Type == "room" || r.Type == "zone" {
			out = append(out, groupInfo{ID: r.ID, Name: r.name(), Kind: r.Type})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ID < out[j].ID
	})
	return out
}

type temperatureState int

const (
	temperatureUnknown temperatureState = iota
	temperatureMixed
	temperatureKnown
)

// groupView is the observed state of one room or zone.
type groupView struct {
	Found        bool
	GroupedLight string
	OnKnown      bool
	On           bool
	BrightKnown  bool
	Brightness   float64
	CTCapable    bool
	MirekMin     int
	MirekMax     int
	Temperature  temperatureState
	Kelvin       float64 // Mean of valid on-light temperatures; meaningful unless unknown.
}

// group derives the observed state of a room or zone.
func (m model) group(id string) groupView {
	var view groupView
	g, ok := m.resources[id]
	if !ok || (g.Type != "room" && g.Type != "zone") {
		return view
	}
	view.Found = true
	for _, service := range g.Services {
		if service.RType == "grouped_light" {
			view.GroupedLight = service.RID
		}
	}
	if grouped, ok := m.resources[view.GroupedLight]; ok {
		if grouped.On != nil {
			view.OnKnown = true
			view.On = grouped.On.On
		}
		if grouped.Dimming != nil {
			view.BrightKnown = true
			view.Brightness = grouped.Dimming.Brightness
		}
	}
	lowest, highest := 0, math.MaxInt
	unionLow, unionHigh := math.MaxInt, 0
	var kelvins []float64
	for _, light := range m.lights(g) {
		ct := light.ColorTemperature
		if ct == nil || ct.Schema == nil {
			continue
		}
		view.CTCapable = true
		lowest = max(lowest, ct.Schema.Minimum)
		highest = min(highest, ct.Schema.Maximum)
		unionLow = min(unionLow, ct.Schema.Minimum)
		unionHigh = max(unionHigh, ct.Schema.Maximum)
		lit := light.On != nil && light.On.On
		if lit && ct.Valid != nil && *ct.Valid && ct.Mirek != nil && *ct.Mirek > 0 {
			kelvins = append(kelvins, mirekToKelvin(*ct.Mirek))
		}
	}
	if view.CTCapable {
		view.MirekMin, view.MirekMax = lowest, highest
		if lowest > highest {
			view.MirekMin, view.MirekMax = unionLow, unionHigh
		}
	}
	if len(kelvins) > 0 {
		low, high, sum := kelvins[0], kelvins[0], 0.0
		for _, k := range kelvins {
			low, high, sum = min(low, k), max(high, k), sum+k
		}
		view.Kelvin = sum / float64(len(kelvins))
		view.Temperature = temperatureMixed
		if high-low <= kelvinStep {
			view.Temperature = temperatureKnown
		}
	}
	return view
}

// lights returns a room's device lights or a zone's lights.
func (m model) lights(g resource) []resource {
	var out []resource
	for _, child := range g.Children {
		switch child.RType {
		case "light":
			if light, ok := m.resources[child.RID]; ok {
				out = append(out, light)
			}
		case "device":
			for _, service := range m.resources[child.RID].Services {
				if service.RType != "light" {
					continue
				}
				if light, ok := m.resources[service.RID]; ok {
					out = append(out, light)
				}
			}
		}
	}
	return out
}

type sceneInfo struct {
	ControlID, SceneID, Label, ShortLabel string
	Active                                bool
}

// scenes lists scenes sorted by label, with stable room-prefixed control IDs.
func (m model) scenes() []sceneInfo {
	var out []sceneInfo
	for _, r := range m.resources {
		if r.Type != "scene" {
			continue
		}
		groupName := ""
		if r.Group != nil {
			groupName = m.resources[r.Group.RID].name()
		}
		info := sceneInfo{
			ControlID:  "hue.scene-" + slug(groupName) + "-" + shortID(r.ID),
			SceneID:    r.ID,
			Label:      strings.TrimSpace(groupName + " " + r.name()),
			ShortLabel: r.name(),
			Active:     r.Status != nil && r.Status.Active != "" && r.Status.Active != "inactive",
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Label != out[j].Label {
			return out[i].Label < out[j].Label
		}
		return out[i].ControlID < out[j].ControlID
	})
	return out
}

// kelvinStep is one temperature dial tick and the agreement tolerance between lights.
const kelvinStep = 100.0

func mirekToKelvin(mirek int) float64 { return 1e6 / float64(mirek) }

// roomScenes lists the scenes of one group sorted by name, as mapped to room slots.
func (m model) roomScenes(groupID string) []sceneInfo {
	var out []sceneInfo
	for _, scene := range m.scenes() {
		if group := m.resources[scene.SceneID].Group; group != nil && group.RID == groupID {
			out = append(out, scene)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ShortLabel != out[j].ShortLabel {
			return out[i].ShortLabel < out[j].ShortLabel
		}
		return out[i].SceneID < out[j].SceneID
	})
	return out
}

// slug lowercases a name into control-ID characters.
func slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if b.Len() > 0 && !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.TrimSuffix(b.String(), "-")
	if out == "" {
		return "group"
	}
	return out
}

// shortID returns the first eight ID-safe characters of a resource UUID.
func shortID(id string) string {
	out := slug(strings.ReplaceAll(id, "-", ""))
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}
