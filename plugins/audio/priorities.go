package audio

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
	"sound-snoofer/internal/windowsaudio"
	"sound-snoofer/snoofer"
)

// priorityEntry is one priority list entry with its current match.
type priorityEntry struct {
	Driver          string `json:",omitempty"`
	Pattern         string `json:",omitempty"`
	ASIOPattern     string `json:",omitempty"`
	PresencePattern string `json:",omitempty"`
	// Entries by identity: the device ID and its label (current name when
	// active). Interfaces name their driver and presence input.
	DeviceID, DeviceName     string           `json:",omitempty"`
	ASIOID, ASIOName         string           `json:",omitempty"`
	PresenceID, PresenceName string           `json:",omitempty"`
	Inputs                   config.MicInputs `json:",omitempty"` // Interface channels by microphone.
	ID                       string           `json:",omitempty"` // Microphone option.
	Matches                  []string         // Presence matches for interfaces.
	Drivers                  []string         `json:",omitempty"` // Interface driver matches.
	InUse                    bool
	Option                   bool `json:",omitempty"` // Microphone currently selectable.
}

// prioritySuggestion is a connected device no entry of a list matches.
type prioritySuggestion struct {
	Name, Driver  string
	ID            string `json:",omitempty"` // Identity, when unique.
	Exact, Device string
}

// priorityView is the Routing screen's model of the editable lists.
type priorityView struct {
	Lists       map[string][]priorityEntry
	Suggestions map[string][]prioritySuggestion
	// Interface additions pair an ASIO driver with a WDM presence input.
	Drivers, Inputs []prioritySuggestion
	Microphones     []string // Every valid microphone priority choice.
	// Mics are the microphones in input order; a device microphone'"'"'s
	// devices are the list ListMicDevices+ID.
	Mics []micView
	// OutputDevices are connected outputs an output slot can use: neither
	// Playback nor another slot.
	OutputDevices []prioritySuggestion
	// Profiles reports whether Auto has a microphone priority; only then is
	// Activity (nil when metering is off) editable.
	Profiles bool
	Activity *config.Activity `json:",omitempty"`
}

// micView is a microphone as the Routing screen shows it.
type micView struct {
	ID, Name      string
	Device        bool // A Windows input device rather than interface channels.
	InUse, Option bool
	Ready         bool // Never metered.
	Silent        bool // Latched silent by activity metering.
}

// suggestion offers a device with generated patterns and, when it is
// unique, its identity: the ASIO driver CLSID, or the one active Windows
// endpoint of its direction with its name.
func suggestion(d model.Device, endpoints []windowsaudio.Endpoint) prioritySuggestion {
	s := prioritySuggestion{Name: d.Name, Driver: d.Driver, Exact: config.ExactPattern(d.Name), Device: config.DevicePattern(d.Name)}
	if d.Driver == "asio" {
		s.ID = d.ID
		return s
	}
	flow := map[string]int{"output": 0, "input": 1}[d.Direction]
	for _, e := range endpoints {
		if e.Flow != flow || e.Name != d.Name {
			continue
		}
		if s.ID != "" {
			s.ID = ""
			break
		}
		s.ID = e.ID
	}
	return s
}

// buildPriorityView reports cfg's lists against the latest observation. It
// uses the planner's matching rules so the editor and routing agree.
func buildPriorityView(cfg config.Config, s control.State) priorityView {
	view := priorityView{Lists: map[string][]priorityEntry{}, Suggestions: map[string][]prioritySuggestion{}}
	cfg = cfg.Resolve(control.DeviceNames(s.DefaultsDetail.Endpoints, s.Snapshot))
	endpoints := s.DefaultsDetail.Endpoints
	studio := cfg.Studio
	if studio == nil {
		return view
	}
	snapshot := s.Snapshot
	var topology *routing.Topology
	if s.Plan != nil {
		topology = s.Plan.Topology
	}
	inventory := model.InventoryDevices(snapshot.Devices)

	interfaces := []priorityEntry{}
	for _, a := range studio.ASIO {
		presence, drivers := routing.InterfaceMatches(a, snapshot)
		inUse := topology != nil && topology.ASIOActive && len(presence) == 1 && slices.Contains(drivers, topology.ASIOName)
		interfaces = append(interfaces, priorityEntry{ASIOPattern: a.ASIOPattern, PresencePattern: a.PresencePattern, ASIOID: a.ASIOID, ASIOName: a.ASIOName, PresenceID: a.PresenceID, PresenceName: a.PresenceName, Inputs: a.Inputs, Matches: presence, Drivers: drivers, InUse: inUse})
	}
	view.Lists[config.ListInterfaces] = interfaces

	playing := ""
	if topology != nil && topology.PlaybackTarget != "" {
		playing = snapshot.Assignments[topology.PlaybackTarget]
		if topology.PlaybackTarget == "A1" && topology.ASIOActive {
			playing = topology.ASIOName
		}
	}
	playback := []priorityEntry{}
	for _, c := range studio.Playback {
		matches := routing.PlaybackMatches(studio, c, snapshot)
		playback = append(playback, priorityEntry{Driver: c.Driver, Pattern: c.Pattern, DeviceID: c.ID, DeviceName: c.Name, Matches: matches, InUse: len(matches) == 1 && matches[0] == playing})
	}
	view.Lists[config.ListPlayback] = playback

	effective := ""
	if topology != nil && topology.Voice != nil {
		effective = topology.Voice.Effective
	}
	// Each device microphone'"'"'s devices are a list; the one on its input is
	// in use.
	deviceMics := []priorityEntry{}
	for n, m := range studio.Mics() {
		view.Mics = append(view.Mics, micView{ID: m.ID, Name: m.Name, Device: m.IsDevice(), Ready: m.Ready, Silent: slices.Contains(s.SilentMics, m.ID), InUse: m.ID == effective, Option: slices.Contains(s.MicOptions, m.ID)})
		view.Microphones = append(view.Microphones, m.ID)
		if !m.IsDevice() {
			continue
		}
		assigned := snapshot.Assignments[fmt.Sprintf("input:%d", n+1)]
		entries := []priorityEntry{}
		for _, c := range m.Devices {
			matches := routing.WebcamMatches(c, snapshot)
			entries = append(entries, priorityEntry{Driver: c.Driver, Pattern: c.Pattern, DeviceID: c.ID, DeviceName: c.Name, Matches: matches, InUse: len(matches) == 1 && matches[0] == assigned})
		}
		view.Lists[config.ListMicDevices+m.ID] = entries
		deviceMics = append(deviceMics, entries...)
	}
	view.Microphones = append(view.Microphones, "off")
	microphones := []priorityEntry{}
	priority := config.DefaultProfiles().Microphones
	if cfg.Profiles != nil {
		priority = cfg.Profiles.Microphones
	}
	for _, id := range priority {
		microphones = append(microphones, priorityEntry{ID: id, Option: slices.Contains(s.MicOptions, id), InUse: id == effective})
	}
	view.Lists[config.ListMicrophones] = microphones
	if cfg.Profiles != nil {
		view.Profiles = true
		if a := cfg.Profiles.Activity; a != nil {
			copied := *a
			view.Activity = &copied
		}
	}

	matched := func(entries []priorityEntry, name string) bool {
		return slices.ContainsFunc(entries, func(e priorityEntry) bool { return slices.Contains(e.Matches, name) })
	}
	for _, d := range inventory {
		switch {
		case d.Driver == "asio" && d.Direction == "output":
			if !slices.ContainsFunc(interfaces, func(e priorityEntry) bool { return slices.Contains(e.Drivers, d.Name) }) {
				view.Drivers = append(view.Drivers, suggestion(d, endpoints))
			}
		case !d.Available || d.Driver != "wdm":
		case d.Direction == "output":
			if d.Name != playing && !studio.OutputDevice(d.Name) {
				view.OutputDevices = append(view.OutputDevices, suggestion(d, endpoints))
			}
			if !matched(playback, d.Name) && !studio.OutputDevice(d.Name) {
				view.Suggestions[config.ListPlayback] = append(view.Suggestions[config.ListPlayback], suggestion(d, endpoints))
			}
		case d.Direction == "input":
			view.Inputs = append(view.Inputs, suggestion(d, endpoints))
			// Inputs no device microphone uses, and no interface'"'"'s companion,
			// are offered to every device microphone and as new ones.
			if !matched(deviceMics, d.Name) && !matched(interfaces, d.Name) {
				view.Suggestions[config.ListMicDevices] = append(view.Suggestions[config.ListMicDevices], suggestion(d, endpoints))
			}
		}
	}
	// The selected interface's ASIO output is a playback choice.
	if topology != nil && topology.ASIOActive && !matched(playback, topology.ASIOName) {
		view.Suggestions[config.ListPlayback] = append(view.Suggestions[config.ListPlayback], suggestion(model.Device{Name: topology.ASIOName, Driver: "asio", ID: asioID(snapshot, topology.ASIOName)}, endpoints))
	}
	return view
}

// priorityControls publishes the view and the edit control. The edit value
// changes with every saved or failed edit, so a GUI waiting on it settles.
func (i *Instance) priorityControls(s control.State) []snoofer.Control {
	i.mu.Lock()
	cfg, raw, editErr := i.running, i.raw, i.editErr
	i.mu.Unlock()
	// Meter ticks publish 20 times a second; devices and matches change far
	// more slowly, so the view is rebuilt after an edit or at most twice a
	// second.
	if now := time.Now(); i.viewData == nil || string(raw) != i.viewRaw || now.Sub(i.viewAt) >= 500*time.Millisecond {
		i.viewData, _ = json.Marshal(buildPriorityView(cfg, s))
		i.viewRaw, i.viewAt = string(raw), now
	}
	data := i.viewData
	return []snoofer.Control{
		{ID: "audio.priorities", Label: "Device priorities", Group: "Routing", Kind: "status", Value: "", Status: editErr, ViewData: data, Available: true},
		{ID: "audio.priority-edit", Label: "Device priority edit", Group: "Routing", Kind: "text", Value: fmt.Sprintf("%x", sha256.Sum256(raw))[:12], Status: editErr,
			Operations: []string{"set"}, Available: i.saveSettings != nil},
		{ID: "audio.output-edit", Label: "Output edit", Group: "Routing", Kind: "text", Value: fmt.Sprintf("%x", sha256.Sum256(raw))[:12], Status: editErr,
			Operations: []string{"set"}, Available: i.saveSettings != nil},
	}
}

// editRequest is a queued settings edit: the edit control and its JSON value.
type editRequest struct{ control, value string }

// queueEdit hands an edit to the edit goroutine; requests must not block
// control dispatch on file writes.
func (i *Instance) queueEdit(ctx context.Context, edit editRequest) error {
	select {
	case i.edits <- edit:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("audio edit queue full")
	}
}

// runEdits owns saving audio settings. Each edit is validated, saved, then
// applied through the worker's reload; failures leave the running
// configuration unchanged and are reported on the edit control.
func (i *Instance) runEdits(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case edit := <-i.edits:
			err := i.applyEdit(edit)
			i.mu.Lock()
			i.editErr = ""
			if err != nil {
				i.editErr = err.Error()
			}
			i.mu.Unlock()
			// A failure shows on the next published state.
			if err != nil {
				continue
			}
			select {
			case i.actions <- control.Reload:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (i *Instance) applyEdit(edit editRequest) error {
	i.mu.Lock()
	settings, raw := i.settings, i.raw
	i.mu.Unlock()
	edited, err := editConfig(settings.Config, edit)
	if err != nil {
		return err
	}
	settings.Config = edited
	next, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	if err := validateSettings(next); err != nil {
		return err
	}
	running, err := i.prepare(settings)
	if err != nil {
		return err
	}
	if err := i.saveSettings("audio", raw, next); err != nil {
		return err
	}
	i.mu.Lock()
	i.settings, i.raw, i.running = settings, next, running
	i.mu.Unlock()
	return nil
}

// editConfig applies a priority or output edit to the audio configuration.
func editConfig(raw json.RawMessage, edit editRequest) ([]byte, error) {
	switch edit.control {
	case "audio.priority-edit":
		var e config.PriorityEdit
		if err := json.Unmarshal([]byte(edit.value), &e); err != nil {
			return nil, fmt.Errorf("invalid edit")
		}
		return config.EditPriorities(raw, e)
	case "audio.output-edit":
		var e config.OutputEdit
		if err := json.Unmarshal([]byte(edit.value), &e); err != nil {
			return nil, fmt.Errorf("invalid edit")
		}
		return config.EditOutputs(raw, e)
	}
	return nil, fmt.Errorf("unknown edit %s", edit.control)
}

// asioID is the CLSID Voicemeeter reports for an ASIO driver.
func asioID(s model.Snapshot, name string) string {
	for _, d := range s.Devices {
		if d.Driver == "asio" && d.Name == name {
			return d.ID
		}
	}
	return ""
}
