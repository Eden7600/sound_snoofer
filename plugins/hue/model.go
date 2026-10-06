package hue

import (
	"encoding/json"
	"fmt"
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

type xyPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type xyColor struct {
	XY xyPoint `json:"xy"`
}

// mirekValue is a white temperature in a scene action or palette; the plugin
// only uses it to draw scene artwork.
type mirekValue struct {
	Mirek *int `json:"mirek"`
}

// sceneAction is what a scene sets on one light.
type sceneAction struct {
	Target reference `json:"target"`
	Action struct {
		On               *onState    `json:"on,omitempty"`
		Dimming          *dimming    `json:"dimming,omitempty"`
		Color            *xyColor    `json:"color,omitempty"`
		ColorTemperature *mirekValue `json:"color_temperature,omitempty"`
		Gradient         *struct {
			Points []struct {
				Color xyColor `json:"color"`
			} `json:"points"`
		} `json:"gradient,omitempty"`
	} `json:"action"`
}

// scenePalette is the color set dynamic and dimming-only scenes are built from.
type scenePalette struct {
	Color []struct {
		Color   xyColor  `json:"color"`
		Dimming *dimming `json:"dimming,omitempty"`
	} `json:"color"`
	ColorTemperature []struct {
		ColorTemperature mirekValue `json:"color_temperature"`
		Dimming          *dimming   `json:"dimming,omitempty"`
	} `json:"color_temperature"`
}

type sceneStatus struct {
	Active string `json:"active"`
}

// UnmarshalJSON reads the scene status object. Other resource types use
// "status" for unrelated values, such as zigbee_connectivity's "connected"
// string, which carry nothing this plugin reads.
func (s *sceneStatus) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || data[0] != '{' {
		*s = sceneStatus{}
		return nil
	}
	type plain sceneStatus
	return json.Unmarshal(data, (*plain)(s))
}

// usedTypes are the resource types the model reads. Malformed resources of
// other types are skipped so unrelated bridge data cannot block loading.
var usedTypes = map[string]bool{"room": true, "zone": true, "device": true, "light": true, "grouped_light": true, "scene": true, "motion": true}

// decodeResources decodes each resource separately, skipping malformed
// resources of unused types and rejecting malformed used ones.
func decodeResources(items []json.RawMessage) ([]resource, error) {
	out := make([]resource, 0, len(items))
	for _, item := range items {
		var r resource
		err := json.Unmarshal(item, &r)
		if err == nil {
			out = append(out, r)
			continue
		}
		var header struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		}
		if json.Unmarshal(item, &header) == nil && !usedTypes[header.Type] {
			continue
		}
		return nil, fmt.Errorf("%s %s: %w", header.Type, header.ID, err)
	}
	return out, nil
}

// resource is the subset of CLIP v2 resources this plugin reads. Event updates
// contain only changed fields, so every field is optional.
type resource struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Metadata *metadata     `json:"metadata,omitempty"`
	Children []reference   `json:"children,omitempty"`
	Services []reference   `json:"services,omitempty"`
	Group    *reference    `json:"group,omitempty"`
	Owner    *reference    `json:"owner,omitempty"`   // Motion services: the sensor device.
	Enabled  *bool         `json:"enabled,omitempty"` // Motion services: the sensor is armed.
	On       *onState      `json:"on,omitempty"`
	Dimming  *dimming      `json:"dimming,omitempty"`
	Status   *sceneStatus  `json:"status,omitempty"`
	Actions  []sceneAction `json:"actions,omitempty"` // Scenes only; used for artwork.
	Palette  *scenePalette `json:"palette,omitempty"` // Scenes only; used for artwork.
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
	if update.Owner != nil {
		r.Owner = update.Owner
	}
	if update.Enabled != nil {
		r.Enabled = update.Enabled
	}
	if update.Dimming != nil {
		r.Dimming = update.Dimming
	}
	if update.Status != nil {
		r.Status = update.Status
	}
	if update.Actions != nil {
		r.Actions = update.Actions
	}
	if update.Palette != nil {
		r.Palette = update.Palette
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

// groupView is the observed state of one room or zone.
type groupView struct {
	Found        bool
	GroupedLight string
	OnKnown      bool
	On           bool
	BrightKnown  bool
	Brightness   float64
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

// motionSensor is one motion service in a room.
type motionSensor struct {
	ID      string
	Known   bool // The bridge reported the enabled flag.
	Enabled bool
}

// motionSensors lists the motion services owned by a room's devices, sorted by
// ID. Zones contain lights rather than devices and have none.
func (m model) motionSensors(groupID string) []motionSensor {
	g, ok := m.resources[groupID]
	if !ok || g.Type != "room" {
		return nil
	}
	var out []motionSensor
	for _, child := range g.Children {
		if child.RType != "device" {
			continue
		}
		for _, service := range m.resources[child.RID].Services {
			if service.RType != "motion" {
				continue
			}
			motion, ok := m.resources[service.RID]
			if !ok {
				continue
			}
			sensor := motionSensor{ID: motion.ID}
			if motion.Enabled != nil {
				sensor.Known, sensor.Enabled = true, *motion.Enabled
			}
			out = append(out, sensor)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
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
