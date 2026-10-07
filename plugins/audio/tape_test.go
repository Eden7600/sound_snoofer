package audio

import (
	"testing"

	"sound-snoofer/internal/control"
	"sound-snoofer/internal/model"
	"sound-snoofer/snoofer"
)

func recorderIn(state string) *model.RecorderSnapshot {
	r := &model.RecorderSnapshot{Values: map[string]float32{}}
	for _, p := range model.RecorderParameters() {
		r.Values[p] = 0
	}
	switch state {
	case "Stopped":
		r.Values["Recorder.stop"] = 1
	case "Playing":
		r.Values["Recorder.play"] = 1
	case "Paused":
		r.Values["Recorder.play"], r.Values["Recorder.pause"] = 1, 1
	case "Recording":
		r.Values["Recorder.record"] = 1
	}
	return r
}

func TestTapeRows(t *testing.T) {
	type want struct {
		value, icon string
		available   [4]bool
	}
	for state, w := range map[string]want{
		"Stopped":   {"Ready", "tape-play", [4]bool{true, false, true, true}},
		"Playing":   {"Playing", "tape-pause", [4]bool{true, true, true, true}},
		"Paused":    {"Paused", "tape-play", [4]bool{true, true, true, true}},
		"Recording": {"Rec", "tape-play", [4]bool{false, false, false, false}},
		"Unknown":   {"N/A", "tape-play", [4]bool{false, false, false, false}},
	} {
		rows := tapeRows(recorderIn(state))
		if rows[0].value != w.value || rows[0].icon != w.icon {
			t.Error(state, rows[0])
		}
		for n, row := range rows {
			if row.available != w.available[n] {
				t.Error(state, row.id, row.available)
			}
		}
	}
	for id, target := range map[string]string{"audio.tape-play": "play", "audio.tape-stop": "stop", "audio.tape-rew": "rew", "audio.tape-ff": "ff"} {
		a, err := action(control.State{}, snoofer.Request{ID: id, Operation: "press"})
		if err != nil || a.Kind != control.Tape || a.Target != target {
			t.Fatal(id, a, err)
		}
	}
}
