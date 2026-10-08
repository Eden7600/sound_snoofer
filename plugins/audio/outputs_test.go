package audio

import (
	"encoding/json"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/routing"
	"sound-snoofer/snoofer"
)

func slotState() control.State {
	s := control.State{Revision: 3, Intent: &config.Intent{Outputs: map[string]map[string]bool{"music": {"virtual:1": true, config.SourceMonitor: false, config.SourceSoundboard: false, config.SourceTape: true}}}}
	s.Plan = &routing.Plan{Topology: &routing.Topology{Outputs: []routing.OutputStatus{{ID: "music", Name: "Music", Device: "Speakers (Arena)", Bus: "A3", State: routing.OutputOK}}}}
	return s
}

func controlByID(t *testing.T, controls []snoofer.Control, id string) snoofer.Control {
	t.Helper()
	for _, c := range controls {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("no control %s", id)
	return snoofer.Control{}
}

func TestSlotControls(t *testing.T) {
	controls := slotControls(slotState())
	header := controlByID(t, controls, "audio.slot-1")
	if header.Label != "Music" || header.Value != "In use" || header.Status != "Speakers (Arena)" || header.Hidden || string(header.ViewData) != `{"ID":"music","Bus":"A3"}` {
		t.Fatal(header)
	}
	if c := controlByID(t, controls, "audio.slot-1:virtual:1"); c.Value != "On" || !c.Available || c.ShortLabel != "Computer" {
		t.Fatal(c)
	}
	if c := controlByID(t, controls, "audio.slot-1:monitor"); c.Value != "Off" || !c.Available {
		t.Fatal(c)
	}
	// Switches the slot does not have, and unconfigured positions, keep their
	// bindings as blank keys.
	for _, id := range []string{"audio.slot-1:virtual:3", "audio.slot-2", "audio.slot-2:monitor", "audio.slot-3:tape"} {
		if c := controlByID(t, controls, id); !c.Hidden || c.Available {
			t.Fatal(c)
		}
	}
}

func TestSlotActionUsesSlotID(t *testing.T) {
	s := slotState()
	a, err := action(s, snoofer.Request{ID: "audio.slot-1:monitor", Operation: "press"})
	if err != nil || a.Kind != control.Edit || a.Row != "slot:music:monitor" || a.Value != "true" {
		t.Fatal(a, err)
	}
	next := s.Intent.Clone()
	if err := control.EditIntent(next, a); err != nil || !next.OutputOn("music", config.SourceMonitor) {
		t.Fatal(err, next.Outputs)
	}
	if _, err := action(s, snoofer.Request{ID: "audio.slot-2:monitor", Operation: "press"}); err == nil {
		t.Fatal("switched a missing slot")
	}
	if err := control.EditIntent(next, control.Action{Row: "slot:gone:monitor", Value: "true"}); err == nil {
		t.Fatal("switched a removed slot")
	}
}

func TestOutputEdits(t *testing.T) {
	i := editInstance(t, func(string, json.RawMessage, json.RawMessage) error { return nil })
	edit := func(e config.OutputEdit) error {
		value, _ := json.Marshal(e)
		return i.applyEdit(editRequest{control: "audio.output-edit", value: string(value)})
	}
	added, _ := json.Marshal(config.Output{Name: "Monitor output", Device: "Speakers (Arena)"})
	if err := edit(config.OutputEdit{Op: "add", Value: string(added)}); err != nil {
		t.Fatal(err)
	}
	outputs := i.running.Studio.Outputs
	if len(outputs) != 1 || outputs[0].ID != "monitor-output" || outputs[0].Device != "Speakers (Arena)" {
		t.Fatal(outputs)
	}
	if err := edit(config.OutputEdit{Op: "rename", ID: "monitor-output", Value: "Monitor"}); err != nil || i.running.Studio.Outputs[0].ID != "monitor-output" {
		t.Fatal("rename must keep the ID", err, i.running.Studio.Outputs)
	}
	if err := edit(config.OutputEdit{Op: "rename", ID: "monitor-output", Value: "A name far too long"}); err == nil {
		t.Fatal("accepted long name")
	}
	if err := edit(config.OutputEdit{Op: "remove", ID: "monitor-output"}); err != nil || len(i.running.Studio.Outputs) != 0 {
		t.Fatal(err, i.running.Studio.Outputs)
	}
	if err := edit(config.OutputEdit{Op: "remove", ID: "monitor-output"}); err == nil {
		t.Fatal("removed a missing output")
	}
}

func TestSilentMicrophoneLabels(t *testing.T) {
	s := control.State{Intent: &config.Intent{Source: "auto", Enabled: true, Mode: "direct", Monitor: "off"}, MicOptions: []string{"desk", "lav", "off"}, SilentMics: []string{"lav"}, ChoiceLabels: map[string]string{"desk": "Desk microphone", "lav": "Lavalier"}}
	source := controlByID(t, controls(s), "audio.normal-source")
	if source.OptionLabels["lav"] != "Lavalier · Silent" || source.OptionLabels["desk"] != "Desk microphone" {
		t.Fatal(source.OptionLabels)
	}
}
