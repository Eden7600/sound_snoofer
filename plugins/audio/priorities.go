package audio

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
	"sound-snoofer/snoofer"
)

// priorityEntry is one priority list entry with its current match.
type priorityEntry struct {
	Driver          string   `json:",omitempty"`
	Pattern         string   `json:",omitempty"`
	ASIOPattern     string   `json:",omitempty"`
	PresencePattern string   `json:",omitempty"`
	Desk, Lav       int      `json:",omitempty"`
	ID              string   `json:",omitempty"` // Microphone option.
	Matches         []string // Presence matches for interfaces.
	Drivers         []string `json:",omitempty"` // Interface driver matches.
	InUse           bool
	Option          bool `json:",omitempty"` // Microphone currently selectable.
}

// prioritySuggestion is a connected device no entry of a list matches.
type prioritySuggestion struct {
	Name, Driver  string
	Exact, Device string
}

// priorityView is the Routing screen's model of the editable lists.
type priorityView struct {
	Lists       map[string][]priorityEntry
	Suggestions map[string][]prioritySuggestion
	// Interface additions pair an ASIO driver with a WDM presence input.
	Drivers, Inputs []string
	Microphones     []string // Every valid microphone option ID.
}

func suggestion(d model.Device) prioritySuggestion {
	return prioritySuggestion{Name: d.Name, Driver: d.Driver, Exact: config.ExactPattern(d.Name), Device: config.DevicePattern(d.Name)}
}

// buildPriorityView reports cfg's lists against the latest observation. It
// uses the planner's matching rules so the editor and routing agree.
func buildPriorityView(cfg config.Config, s control.State) priorityView {
	view := priorityView{Lists: map[string][]priorityEntry{}, Suggestions: map[string][]prioritySuggestion{}, Microphones: []string{"desk", "lav", "webcam", "off"}}
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
		interfaces = append(interfaces, priorityEntry{ASIOPattern: a.ASIOPattern, PresencePattern: a.PresencePattern, Desk: a.Inputs[0], Lav: a.Inputs[1], Matches: presence, Drivers: drivers, InUse: inUse})
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
		playback = append(playback, priorityEntry{Driver: c.Driver, Pattern: c.Pattern, Matches: matches, InUse: len(matches) == 1 && matches[0] == playing})
	}
	view.Lists[config.ListPlayback] = playback

	webcam := []priorityEntry{}
	assigned := snapshot.Assignments["input:3"]
	for _, c := range studio.FallbackMic {
		matches := routing.WebcamMatches(c, snapshot)
		webcam = append(webcam, priorityEntry{Driver: c.Driver, Pattern: c.Pattern, Matches: matches, InUse: len(matches) == 1 && matches[0] == assigned})
	}
	view.Lists[config.ListWebcam] = webcam

	effective := ""
	if topology != nil && topology.Voice != nil {
		effective = topology.Voice.Effective
	}
	microphones := []priorityEntry{}
	priority := config.DefaultProfiles().Microphones
	if cfg.Profiles != nil {
		priority = cfg.Profiles.Microphones
	}
	for _, id := range priority {
		microphones = append(microphones, priorityEntry{ID: id, Option: slices.Contains(s.MicOptions, id), InUse: id == effective})
	}
	view.Lists[config.ListMicrophones] = microphones

	matched := func(entries []priorityEntry, name string) bool {
		return slices.ContainsFunc(entries, func(e priorityEntry) bool { return slices.Contains(e.Matches, name) })
	}
	for _, d := range inventory {
		switch {
		case d.Driver == "asio" && d.Direction == "output":
			if !slices.ContainsFunc(interfaces, func(e priorityEntry) bool { return slices.Contains(e.Drivers, d.Name) }) {
				view.Drivers = append(view.Drivers, d.Name)
			}
		case !d.Available || d.Driver != "wdm":
		case d.Direction == "output":
			if !matched(playback, d.Name) {
				view.Suggestions[config.ListPlayback] = append(view.Suggestions[config.ListPlayback], suggestion(d))
			}
		case d.Direction == "input":
			view.Inputs = append(view.Inputs, d.Name)
			// An interface's companion input is not a webcam candidate.
			if !matched(webcam, d.Name) && !matched(interfaces, d.Name) {
				view.Suggestions[config.ListWebcam] = append(view.Suggestions[config.ListWebcam], suggestion(d))
			}
		}
	}
	// The selected interface's ASIO output is a playback choice.
	if topology != nil && topology.ASIOActive && !matched(playback, topology.ASIOName) {
		view.Suggestions[config.ListPlayback] = append(view.Suggestions[config.ListPlayback], suggestion(model.Device{Name: topology.ASIOName, Driver: "asio"}))
	}
	return view
}

// priorityControls publishes the view and the edit control. The edit value
// changes with every saved or failed edit, so a GUI waiting on it settles.
func (i *Instance) priorityControls(s control.State) []snoofer.Control {
	i.mu.Lock()
	cfg, raw, editErr := i.running, i.raw, i.editErr
	i.mu.Unlock()
	data, _ := json.Marshal(buildPriorityView(cfg, s))
	return []snoofer.Control{
		{ID: "audio.priorities", Label: "Device priorities", Group: "Routing", Kind: "status", Value: "", Status: editErr, ViewData: data, Available: true},
		{ID: "audio.priority-edit", Label: "Device priority edit", Group: "Routing", Kind: "text", Value: fmt.Sprintf("%x", sha256.Sum256(raw))[:12], Status: editErr,
			Operations: []string{"set"}, Available: i.saveSettings != nil},
	}
}

// queueEdit hands an edit to the edit goroutine; requests must not block
// control dispatch on file writes.
func (i *Instance) queueEdit(ctx context.Context, value string) error {
	select {
	case i.edits <- value:
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
		case value := <-i.edits:
			err := i.applyEdit(value)
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

func (i *Instance) applyEdit(value string) error {
	var e config.PriorityEdit
	if err := json.Unmarshal([]byte(value), &e); err != nil {
		return fmt.Errorf("invalid edit")
	}
	i.mu.Lock()
	settings, raw := i.settings, i.raw
	i.mu.Unlock()
	edited, err := config.EditPriorities(settings.Config, e)
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
