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
		available   [4]bool // Play, Stop, Rew, FF.
	}
	recPaused := recorderIn("Recording")
	recPaused.Values["Recorder.pause"] = 1
	for name, c := range map[string]struct {
		r      *model.RecorderSnapshot
		paused bool
		want   want
	}{
		"Stopped":          {recorderIn("Stopped"), false, want{"Ready", "tape-play", [4]bool{true, false, true, true}}},
		"Tape paused":      {recorderIn("Stopped"), true, want{"Paused", "tape-play", [4]bool{true, true, true, true}}},
		"Playing":          {recorderIn("Playing"), false, want{"Playing", "tape-pause", [4]bool{true, true, true, true}}},
		"Playback paused":  {recorderIn("Paused"), false, want{"Paused", "tape-play", [4]bool{true, true, true, true}}},
		"Recording":        {recorderIn("Recording"), false, want{"Rec", "tape-pause", [4]bool{true, true, false, false}}},
		"Recording paused": {recPaused, false, want{"Paused", "tape-play", [4]bool{true, true, false, false}}},
		"Unknown":          {recorderIn("Unknown"), false, want{"N/A", "tape-play", [4]bool{false, false, false, false}}},
	} {
		rows := tapeRows(c.r, c.paused)
		if rows[0].value != c.want.value || rows[0].icon != c.want.icon {
			t.Error(name, rows[0])
		}
		for n, row := range rows {
			if row.available != c.want.available[n] {
				t.Error(name, row.id, row.available)
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

func TestRecordKeyByRecorderState(t *testing.T) {
	recPaused := recorderIn("Recording")
	recPaused.Values["Recorder.pause"] = 1
	for name, c := range map[string]struct {
		r    *model.RecorderSnapshot
		want control.ActionKind
	}{
		"Stopped":          {recorderIn("Stopped"), control.RecordStart},
		"Playing":          {recorderIn("Playing"), control.RecordStart},
		"Playback paused":  {recorderIn("Paused"), control.RecordStart},
		"Recording":        {recorderIn("Recording"), control.RecordStop},
		"Recording paused": {recPaused, control.RecordStop},
	} {
		a, err := action(control.State{Recorder: c.r}, snoofer.Request{ID: "audio.record-toggle", Operation: "press"})
		if err != nil || a.Kind != c.want {
			t.Error(name, a.Kind, err)
		}
	}
	if _, err := action(control.State{Recorder: recorderIn("Unknown")}, snoofer.Request{ID: "audio.record-toggle", Operation: "press"}); err == nil {
		t.Fatal("unknown recorder state accepted")
	}
}
