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
					inputs[0], inputs[1] = true, true
				}
			}
		}
		for patch := 0; patch < 4; patch++ {
			if op.Parameter == fmt.Sprintf("Patch.asio[%d]", patch) {
				inputs[patch/2] = true
			}
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
