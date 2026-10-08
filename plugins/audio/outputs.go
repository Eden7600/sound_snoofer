package audio

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/routing"
	"sound-snoofer/snoofer"
)

// slotSources are the per-slot switches in deck column order; switches a
// slot does not have are published hidden so deck bindings stay in place.
var slotSources = []string{"virtual:1", "virtual:2", "virtual:3", config.SourceMonitor, config.SourceSoundboard, config.SourceTape}

var sourceLabels = map[string]string{"virtual:1": "Computer", "virtual:2": "Virtual 2", "virtual:3": "Virtual 3",
	config.SourceMonitor: "Monitor", config.SourceSoundboard: "Soundboard", config.SourceTape: "Tape"}

var sourceIcons = map[string]string{"virtual:1": "record-computer", "virtual:2": "record-computer", "virtual:3": "record-computer",
	config.SourceMonitor: "monitor", config.SourceSoundboard: "soundboard-play", config.SourceTape: "tape-play"}

var outputStates = map[string]string{routing.OutputOK: "In use", routing.OutputMissing: "Missing", routing.OutputNoOutput: "No output"}

func planOutputs(s control.State) []routing.OutputStatus {
	if s.Plan == nil || s.Plan.Topology == nil {
		return nil
	}
	return s.Plan.Topology.Outputs
}

// slotControls publishes each output slot by position, with its state and a
// switch per source. Positions without a slot are hidden, not removed, so
// deck bindings keep their keys.
func slotControls(s control.State) []snoofer.Control {
	outputs := planOutputs(s)
	out := []snoofer.Control{}
	for n := 1; n <= config.MaxOutputs; n++ {
		id := fmt.Sprintf("audio.slot-%d", n)
		header := snoofer.Control{ID: id, Label: fmt.Sprintf("Output %d", n), Group: "Routing", Kind: "status", Icon: "vr-playback"}
		var slot *routing.OutputStatus
		if n <= len(outputs) {
			slot = &outputs[n-1]
			header.Label, header.ShortLabel, header.Value = slot.Name, slot.Name, outputStates[slot.State]
			header.Status = slot.Device
			header.Available = true
			// The GUI addresses edits by ID, never by position.
			header.ViewData, _ = json.Marshal(struct{ ID, Bus string }{slot.ID, slot.Bus})
		}
		header.Hidden = slot == nil
		out = append(out, header)
		for _, source := range slotSources {
			c := snoofer.Control{ID: id + ":" + source, Label: header.Label + " " + sourceLabels[source], ShortLabel: sourceLabels[source], Group: "Routing", Kind: "toggle",
				Value: "Off", Operations: []string{"press", "set"}, Options: []string{"On", "Off"}, Icon: sourceIcons[source]}
			if slot != nil && s.Intent != nil {
				on, exists := s.Intent.Outputs[slot.ID][source]
				c.Available = exists
				if on {
					c.Value = "On"
				}
			}
			c.Hidden = !c.Available
			out = append(out, c)
		}
	}
	return out
}

// slotAction turns a slot switch request into a saved-choice edit addressed
// by the slot's ID, so a slot reordered meanwhile is never switched instead.
func slotAction(s control.State, key string, r snoofer.Request, a control.Action) (control.Action, error) {
	position, source, _ := strings.Cut(strings.TrimPrefix(key, "slot-"), ":")
	n, err := strconv.Atoi(position)
	outputs := planOutputs(s)
	if err != nil || n < 1 || n > len(outputs) || s.Intent == nil {
		return a, fmt.Errorf("output unavailable")
	}
	slot := outputs[n-1]
	on, exists := s.Intent.Outputs[slot.ID][source]
	if !exists {
		return a, fmt.Errorf("output switch unavailable")
	}
	switch r.Operation {
	case "press":
		on = !on
	case "set":
		on = r.Value == "On"
	default:
		return a, fmt.Errorf("unsupported operation")
	}
	a.Row, a.Value = "slot:"+slot.ID+":"+source, strconv.FormatBool(on)
	return a, nil
}
