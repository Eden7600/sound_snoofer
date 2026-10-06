package hue

import (
	"encoding/json"
	"slices"
	"strings"

	"sound-snoofer/snoofer"
)

// chosen reports whether Snoofer controls a room or zone. An empty list
// means no choice has been made, so every group is controlled.
func (s Settings) chosen(groupID string) bool {
	return len(s.Rooms) == 0 || slices.Contains(s.Rooms, groupID)
}

// chosenGroups lists the existing groups Snoofer controls, by name.
func (w *worker) chosenGroups() []groupInfo {
	var out []groupInfo
	for _, g := range w.model.groups() {
		if w.settings.chosen(g.ID) {
			out = append(out, g)
		}
	}
	return out
}

// enforceRoom moves the selection to the first chosen group when the selected
// group is not chosen. It needs bridge data, so it only acts while connected.
func (w *worker) enforceRoom() {
	if !w.connected || len(w.settings.Rooms) == 0 || slices.Contains(w.settings.Rooms, w.settings.Group) {
		return
	}
	if groups := w.chosenGroups(); len(groups) > 0 {
		w.selectGroup(groups[0].ID)
	}
}

// setRooms saves the comma-separated list of chosen groups. At least one
// existing group must stay chosen.
func (w *worker) setRooms(value string) {
	known := map[string]bool{}
	for _, g := range w.model.groups() {
		known[g.ID] = true
	}
	var rooms []string
	for _, id := range strings.Split(value, ",") {
		id = strings.TrimSpace(id)
		if known[id] && !slices.Contains(rooms, id) {
			rooms = append(rooms, id)
		}
	}
	if len(rooms) == 0 {
		w.roomsErr = "Choose at least one room"
		return
	}
	next := w.settings
	next.Rooms = rooms
	if err := w.save(next); err != nil {
		w.roomsErr = err.Error()
		return
	}
	w.roomsErr = ""
	w.enforceRoom()
}

// roomChoice is one row of the Lights screen's Rooms card.
type roomChoice struct {
	ID, Name, Kind string
	Chosen         bool
}

// roomsControl edits the chosen groups. It is a text control so the deck never
// offers it as a binding.
func (w *worker) roomsControl() snoofer.Control {
	var choices []roomChoice
	for _, g := range w.model.groups() {
		choices = append(choices, roomChoice{ID: g.ID, Name: g.Name, Kind: g.Kind, Chosen: w.settings.chosen(g.ID)})
	}
	view, _ := json.Marshal(choices) // Plain string and bool fields always marshal.
	return snoofer.Control{ID: "hue.rooms", Label: "Hue rooms", Group: "Hue", Kind: "text", Value: strings.Join(w.settings.Rooms, ","),
		Status: w.roomsErr, ViewData: view, Operations: []string{"set"}, Available: w.services.Live && w.connected}
}
