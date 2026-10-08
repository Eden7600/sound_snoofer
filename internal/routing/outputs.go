package routing

import (
	"fmt"
	"slices"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

// Output slot states.
const (
	OutputOK       = "ok"        // The device is available on its bus.
	OutputMissing  = "missing"   // The device is unavailable; any held bus stays reserved.
	OutputNoOutput = "no-output" // No bus is free for the device.
)

// OutputStatus is an output slot's resolved destination.
type OutputStatus struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Device  string   `json:"device"`
	Bus     string   `json:"bus,omitempty"`
	State   string   `json:"state"`
	Sources []string `json:"sources,omitempty"` // Switched-on sources.
}

// Receives reports whether the slot's source switch is on.
func (o OutputStatus) Receives(source string) bool { return slices.Contains(o.Sources, source) }

// OutputAt returns the slot holding bus, if any.
func (t *Topology) OutputAt(bus string) *OutputStatus {
	for n := range t.Outputs {
		if bus != "" && t.Outputs[n].Bus == bus {
			return &t.Outputs[n]
		}
	}
	return nil
}

// OutputBuses lists the buses of slots receiving source.
func (t *Topology) OutputBuses(source string) []string {
	buses := []string{}
	for _, o := range t.Outputs {
		if o.Bus != "" && o.Receives(source) {
			buses = append(buses, o.Bus)
		}
	}
	return buses
}

// outputSlots tracks which A buses output slot devices hold.
type outputSlots struct {
	profile    *config.Studio
	snapshot   model.Snapshot
	buses      int
	asioActive bool
	held       map[string]string // Output ID to the bus holding its device.
	evicted    map[string]bool   // Slots whose bus Playback took.
	duplicates []string          // Further buses holding a slot device; cleared.
}

// holdOutputBuses reserves each bus that already holds a slot device, so a
// slot keeps its bus while its device is disconnected and never reshuffles.
func holdOutputBuses(profile *config.Studio, s model.Snapshot, buses int, asioActive bool) *outputSlots {
	slots := &outputSlots{profile: profile, snapshot: s, buses: buses, asioActive: asioActive, held: map[string]string{}, evicted: map[string]bool{}}
	for i := 1; i <= buses; i++ {
		bus := fmt.Sprintf("A%d", i)
		if asioActive && i == 1 {
			continue // ASIO replaces a slot device found on A1.
		}
		for _, o := range profile.Outputs {
			if s.Assignments[bus] != o.Device {
				continue
			}
			if slots.held[o.ID] == "" {
				slots.held[o.ID] = bus
			} else {
				slots.duplicates = append(slots.duplicates, bus)
			}
		}
	}
	return slots
}

// evict gives Playback the bus of the last slot holding one, when no other
// bus is free.
func (slots *outputSlots) evict() string {
	for n := len(slots.profile.Outputs) - 1; n >= 0; n-- {
		id := slots.profile.Outputs[n].ID
		if bus := slots.held[id]; bus != "" {
			delete(slots.held, id)
			slots.evicted[id] = true
			return bus
		}
	}
	return ""
}

// place resolves every slot in configuration order: its held bus, else the
// lowest free bus. A free bus is empty or holds a former playback device.
func (slots *outputSlots) place(c config.Config, t *Topology, ownsPlayback func(string) bool) {
	intent := c.VoiceIntent()
	taken := map[string]bool{t.PlaybackTarget: true}
	for _, bus := range slots.held {
		taken[bus] = true
	}
	for _, o := range slots.profile.Outputs {
		status := OutputStatus{ID: o.ID, Name: o.Name, Device: o.Device, Bus: slots.held[o.ID]}
		for _, source := range slots.profile.OutputSources() {
			on := slices.Contains(o.Sources, source)
			if intent != nil {
				on = intent.OutputOn(o.ID, source)
			}
			if on {
				status.Sources = append(status.Sources, source)
			}
		}
		available := slices.ContainsFunc(slots.snapshot.Devices, func(d model.Device) bool {
			return d.Available && d.Direction == "output" && d.Driver == "wdm" && d.Name == o.Device
		})
		switch {
		case slots.evicted[o.ID]:
			status.State = OutputNoOutput
		case !available:
			status.State = OutputMissing
		case status.Bus != "":
			status.State = OutputOK
		default:
			status.State = OutputNoOutput
			for i := 1; i <= slots.buses; i++ {
				bus := fmt.Sprintf("A%d", i)
				current := slots.snapshot.Assignments[bus]
				if taken[bus] || (slots.asioActive && i == 1) || (current != "" && !ownsPlayback(current)) {
					continue
				}
				status.Bus, status.State = bus, OutputOK
				taken[bus] = true
				break
			}
		}
		t.Outputs = append(t.Outputs, status)
	}
}

func boolValue(on bool) int {
	if on {
		return 1
	}
	return 0
}
