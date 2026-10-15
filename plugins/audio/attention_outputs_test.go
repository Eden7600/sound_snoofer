package audio

import (
	"testing"

	"sound-snoofer/internal/routing"
	"sound-snoofer/snoofer"
)

func TestOutputAttentionDistinguishesAllocationFromAbsence(t *testing.T) {
	for _, state := range []string{routing.OutputOK, routing.OutputEmpty, routing.OutputMissing, routing.OutputNoOutput} {
		for _, bus := range []string{"", "A3"} {
			t.Run(state+bus, func(t *testing.T) {
				value := slotState()
				value.Plan.Topology.Outputs[0].State = state
				value.Plan.Topology.Outputs[0].Bus = bus
				controls := slotControls(value)
				want := snoofer.AttentionLevel("")
				if state == routing.OutputNoOutput {
					want = snoofer.AttentionWarning
				}
				if got := controlByID(t, controls, "audio.slot-1").Attention; got != want {
					t.Fatalf("%s with bus %q: %q, want %q", state, bus, got, want)
				}
				if got := controlByID(t, controls, "audio.slot-2").Attention; got != "" {
					t.Fatalf("unconfigured slot marked: %q", got)
				}
			})
		}
	}
}
