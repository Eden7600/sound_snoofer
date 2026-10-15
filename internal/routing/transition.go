package routing

import (
	"fmt"
	"strings"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

// buildVoiceTransition gates only changed sends and hardware dependencies.
// The full desired operations remain available for controller drift checks.
func buildVoiceTransition(c config.Config, s model.Snapshot, t *Topology) {
	inputs := map[int]bool{}
	outputs := map[string]bool{}
	// Interface microphones own the two patch cells of their strip: Patch.asio[k] feeds strip k/2.
	patchStrip := func(op Operation) (int, bool) {
		var cell int
		if _, err := fmt.Sscanf(op.Parameter, "Patch.asio[%d]", &cell); err != nil {
			return 0, false
		}
		return cell / 2, true
	}
	for _, op := range t.Operations {
		if !op.Change {
			continue
		}
		if op.Device != nil {
			slot, err := model.ParseSlot(op.Target)
			if err != nil {
				continue // Targets have already been validated by the planner.
			}
			if slot.Direction == "input" {
				inputs[slot.Index] = true
			} else {
				outputs[op.Target] = true
				if op.Target == "A1" && (t.ASIOActive || c.Studio.OwnsASIO(op.BeforeName)) {
					// Every interface microphone hears the A1 device.
					for _, patch := range t.Operations {
						if strip, ok := patchStrip(patch); ok {
							inputs[strip] = true
						}
					}
				}
			}
		}
		if strip, ok := patchStrip(op); ok {
			inputs[strip] = true
		}
	}
	// Hardware changes upstream of Element also affect its return strip.
	for strip := 0; strip < 5; strip++ {
		if !inputs[strip] {
			continue
		}
		param := fmt.Sprintf("Strip[%d].B2", strip)
		if s.Numbers[param] != 0 {
			inputs[6] = true
		}
		for _, op := range t.Operations {
			if op.Parameter == param && op.Value != 0 {
				inputs[6] = true
			}
		}
	}
	gated := func(op Operation) bool {
		if !strings.HasPrefix(op.Parameter, "Strip[") && !strings.HasPrefix(op.Parameter, "Recorder.A") && !strings.HasPrefix(op.Parameter, "Recorder.B") {
			return false
		}
		if op.Change {
			return true
		}
		for strip := range inputs {
			if strings.HasPrefix(op.Parameter, fmt.Sprintf("Strip[%d].", strip)) {
				return true
			}
		}
		for output := range outputs {
			if strings.HasSuffix(op.Parameter, "."+output) {
				return true
			}
		}
		return false
	}
	changed := false
	for _, op := range t.Operations {
		changed = changed || op.Change
		if gated(op) && op.BeforeValue != 0 {
			off := op
			off.Value, off.Change = 0, true
			t.Transition = append(t.Transition, off)
		}
	}
	if !changed {
		return
	}
	for _, op := range t.Operations {
		if !gated(op) {
			t.Transition = append(t.Transition, op)
		}
	}
	for _, op := range t.Operations {
		if gated(op) {
			op.BeforeValue = 0
			op.Change = op.Value != 0
			t.Transition = append(t.Transition, op)
		}
	}
}
