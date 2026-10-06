package hue

import (
	"encoding/json"
	"slices"
	"testing"

	"sound-snoofer/snoofer"
)

func (h *harness) savedSettings() Settings {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.saved
}

func TestRoomScope(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	h := startHarness(t, bridge, paired(bridge, "room-1"), true)

	// No choice yet: every group is controlled and the Room key can switch.
	group := h.waitControl("hue.group", func(c snoofer.Control) bool { return len(c.Options) == 2 })
	if group.Hidden {
		t.Fatal("room key hidden with two rooms")
	}
	rooms := h.waitControl("hue.rooms", func(c snoofer.Control) bool { return c.Available })
	var choices []roomChoice
	if err := json.Unmarshal(rooms.ViewData, &choices); err != nil || len(choices) != 2 || !choices[0].Chosen || !choices[1].Chosen {
		t.Fatalf("choices %s %v", rooms.ViewData, err)
	}

	// Choosing only the zone moves the selection there and hides the rest.
	h.mustDispatch("hue.rooms", "set", 0, "zone-1, unknown")
	group = h.waitControl("hue.group", func(c snoofer.Control) bool { return c.Value == "zone-1" })
	if !slices.Equal(group.Options, []string{"zone-1"}) || !group.Hidden || group.OptionLabels["zone-1"] != "Desk (zone)" {
		t.Fatalf("restricted group %+v", group)
	}
	if saved := h.savedSettings(); !slices.Equal(saved.Rooms, []string{"zone-1"}) || saved.Group != "zone-1" {
		t.Fatalf("saved %+v", saved)
	}
	h.waitControl("hue.scene-desk-8b7e0d21", func(c snoofer.Control) bool { return c.Available })
	if _, ok := h.find("hue.scene-studio-3f2a9c10"); ok {
		t.Fatal("scene of an unchosen room published")
	}

	// The last room stays chosen, and unchosen rooms cannot be selected.
	h.mustDispatch("hue.rooms", "set", 0, "")
	h.waitControl("hue.rooms", func(c snoofer.Control) bool { return c.Status == "Choose at least one room" })
	if err := h.dispatch("hue.group", "set", 0, "room-1"); err == nil {
		t.Fatal("unchosen room selected")
	}
	if saved := h.savedSettings(); !slices.Equal(saved.Rooms, []string{"zone-1"}) {
		t.Fatalf("rejected edits changed settings %+v", saved)
	}
}

func TestRoomScopeMovesStaleSelection(t *testing.T) {
	bridge := newFakeBridge(t, "b1", studio()...)
	settings := paired(bridge, "zone-1")
	settings.Rooms = []string{"room-1"}
	h := startHarness(t, bridge, settings, true)
	h.waitControl("hue.group", func(c snoofer.Control) bool { return c.Value == "room-1" })
	if saved := h.savedSettings(); saved.Group != "room-1" {
		t.Fatalf("selection not saved %+v", saved)
	}
}
