package routing

import (
	"fmt"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

func addRehearsal(c config.Config, s model.Snapshot, t *Topology) error {
	i := c.VoiceIntent()
	r := i.Recording
	add := func(param string, value int) error {
		before, ok := s.Recorder.Values[param]
		if !ok {
			return fmt.Errorf("snapshot missing %s", param)
		}
		t.Operations = append(t.Operations, Operation{Parameter: param, Value: value, BeforeValue: before, Change: before != float32(value)})
		return nil
	}
	b2, loop := 0, 0
	if r.ToVST {
		b2 = 1
	}
	if r.Loop {
		loop = 1
	}
	if err := add("Recorder.B2", b2); err != nil {
		return err
	}
	if err := add("Recorder.mode.Loop", loop); err != nil {
		return err
	}
	// Tape A sends are owned for rehearsal and cleared when leaving it.
	if r.ToVST || r.TapeRoutingManaged || s.Recorder.Values["Recorder.B2"] != 0 {
		for _, bus := range []string{"A1", "A2", "A3", "A4", "A5", "B1", "B3"} {
			value := 0
			if r.ToVST && i.Monitor == "pre" && t.PlaybackTarget == bus {
				value = 1
			}
			if err := add("Recorder."+bus, value); err != nil {
				return err
			}
		}
	}
	return nil
}
