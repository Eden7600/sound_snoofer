package audio

import (
	"testing"

	"sound-snoofer/internal/config"
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

func TestIdleTransportKeysReadReady(t *testing.T) {
	intent := &config.Intent{Enabled: true, Source: "desk", Mode: "direct", Monitor: "off", Recording: &config.RecordingChoices{}}
	s := control.State{Connected: true, Live: true, Intent: intent, ActiveIntent: intent.Clone(), Recorder: recorderIn("Stopped"), TapePaused: true}
	byID := map[string]snoofer.Control{}
	for _, c := range controls(s) {
		byID[c.ID] = c
		// Only selections label their options; an empty value elsewhere must
		// never read Automatic.
		if c.Kind != "selection" && c.OptionLabels[c.Value] != "" {
			t.Errorf("%s (%s) shows option label %q for value %q", c.ID, c.Kind, c.OptionLabels[c.Value], c.Value)
		}
	}
	for _, id := range []string{"audio.tape-stop", "audio.tape-rew", "audio.tape-ff"} {
		c, ok := byID[id]
		if !ok || c.Value != "Ready" || !c.Available {
			t.Errorf("%s: value %q available %v", id, c.Value, c.Available)
		}
	}
	if byID["audio.source"].OptionLabels["auto"] != "Automatic" {
		t.Fatal("selection lost its Automatic label")
	}
}
