package audio

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
	"sound-snoofer/snoofer"
)

func priorityState() control.State {
	device := func(name, direction, driver string, available bool) model.Device {
		return model.Device{Name: name, Direction: direction, Driver: driver, Available: available}
	}
	return control.State{
		MicOptions: []string{"desk", "lav", "webcam", "off"},
		Snapshot: model.Snapshot{
			Devices: []model.Device{
				device("Universal Audio Volt", "output", "asio", false),
				device("Focusrite USB ASIO", "output", "asio", false),
				device("INPUT 1/2 (Volt 2)", "input", "wdm", true),
				device("Headphones (AirPods Pro)", "output", "wdm", true),
				device("Speakers (SteelSeries Arena 7)", "output", "wdm", true),
				device("Game (SteelSeries Arena 7)", "output", "wdm", true),
				device("Speakers (Realtek Audio)", "output", "wdm", true),
				device("Microphone (Insta360 Link 2)", "input", "wdm", true),
				device("Microphone (Realtek Audio)", "input", "wdm", true),
			},
			Assignments: map[string]string{"A1": "Universal Audio Volt", "A2": "Headphones (AirPods Pro)", "input:3": "Microphone (Insta360 Link 2)"},
		},
		Plan: &routing.Plan{Topology: &routing.Topology{ASIOActive: true, ASIOName: "Universal Audio Volt", PlaybackTarget: "A2", Voice: &routing.VoiceStatus{Effective: "lav"}}},
	}
}

func defaultConfig(t *testing.T) config.Config {
	t.Helper()
	c, err := config.Decode(config.DefaultBytes())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPriorityView(t *testing.T) {
	view := buildPriorityView(defaultConfig(t), priorityState())
	interfaces := view.Lists[config.ListInterfaces]
	if len(interfaces) != 1 || !interfaces[0].InUse || !slices.Equal(interfaces[0].Matches, []string{"INPUT 1/2 (Volt 2)"}) || interfaces[0].Desk != 1 || interfaces[0].Lav != 2 {
		t.Fatal(interfaces)
	}
	playback := view.Lists[config.ListPlayback]
	if !playback[0].InUse || len(playback[1].Matches) != 2 || playback[1].InUse || playback[2].InUse || !slices.Equal(playback[2].Matches, []string{"Universal Audio Volt"}) {
		t.Fatal("playback matches", playback)
	}
	if webcam := view.Lists[config.ListWebcam]; !webcam[0].InUse {
		t.Fatal(webcam)
	}
	mics := view.Lists[config.ListMicrophones]
	if mics[0].ID != "lav" || !mics[0].InUse || !mics[0].Option || mics[1].InUse {
		t.Fatal(mics)
	}
	names := func(list []prioritySuggestion) []string {
		out := []string{}
		for _, s := range list {
			out = append(out, s.Name)
		}
		return out
	}
	if got := names(view.Suggestions[config.ListPlayback]); !slices.Equal(got, []string{"Speakers (Realtek Audio)"}) {
		t.Fatal("playback suggestions", got)
	}
	// The interface's companion input is not offered as a webcam.
	if got := names(view.Suggestions[config.ListWebcam]); !slices.Equal(got, []string{"Microphone (Realtek Audio)"}) {
		t.Fatal("webcam suggestions", got)
	}
	s := view.Suggestions[config.ListWebcam][0]
	if s.Exact != `(?i)^Microphone \(Realtek Audio\)$` || s.Device != `(?i)Realtek Audio` {
		t.Fatal(s)
	}
	if !slices.Equal(names(view.Drivers), []string{"Focusrite USB ASIO"}) || !slices.Contains(names(view.Inputs), "INPUT 1/2 (Volt 2)") || view.Drivers[0].Exact != `(?i)^Focusrite USB ASIO$` {
		t.Fatal(view.Drivers, view.Inputs)
	}
}

// An installed driver without its presence input is never in use.
func TestPriorityViewDriverWithoutHardware(t *testing.T) {
	s := priorityState()
	s.Snapshot.Devices = slices.DeleteFunc(s.Snapshot.Devices, func(d model.Device) bool { return d.Name == "INPUT 1/2 (Volt 2)" })
	s.Plan.Topology.ASIOActive = false
	if entry := buildPriorityView(defaultConfig(t), s).Lists[config.ListInterfaces][0]; entry.InUse || len(entry.Matches) != 0 || len(entry.Drivers) != 1 {
		t.Fatal(entry)
	}
}

func editInstance(t *testing.T, save func(string, json.RawMessage, json.RawMessage) error) *Instance {
	t.Helper()
	settings := Settings{Config: config.DefaultBytes(), StatePath: "audio"}
	i := &Instance{actions: make(chan control.Action, 1), edits: make(chan string, 1), settings: settings, raw: snoofer.MarshalSettings(settings), saveSettings: save}
	i.prepare = func(s Settings) (config.Config, error) { return config.Decode(s.Config) }
	var err error
	if i.running, err = i.prepare(settings); err != nil {
		t.Fatal(err)
	}
	return i
}

func TestPriorityEditSavesThenReloads(t *testing.T) {
	var saved json.RawMessage
	var expected json.RawMessage
	i := editInstance(t, func(id string, before, next json.RawMessage) error {
		expected, saved = before, next
		return nil
	})
	original := i.raw
	value, _ := json.Marshal(config.PriorityEdit{List: config.ListPlayback, Op: "move", Index: 1, Value: "up"})
	if err := i.applyEdit(string(value)); err != nil {
		t.Fatal(err)
	}
	if string(expected) != string(original) || saved == nil {
		t.Fatal("save must be revision-checked against the loaded settings")
	}
	var s Settings
	if err := json.Unmarshal(saved, &s); err != nil || s.StatePath != "audio" {
		t.Fatal("saved settings must keep the relative state path", s.StatePath, err)
	}
	if i.running.Studio.Playback[0].Pattern != "(?i)steelseries.*arena" || string(i.raw) != string(saved) {
		t.Fatal("running configuration not replaced", i.running.Studio.Playback)
	}
}

func TestRefusedPriorityEdits(t *testing.T) {
	saves := 0
	failSave := false
	i := editInstance(t, func(string, json.RawMessage, json.RawMessage) error {
		saves++
		if failSave {
			return errors.New("plugin settings changed; reload before saving")
		}
		return nil
	})
	before := i.running.Studio.Playback[0].Pattern
	bad, _ := json.Marshal(config.PriorityEdit{List: config.ListPlayback, Op: "set", Field: "pattern", Value: "(unclosed"})
	if err := i.applyEdit(string(bad)); err == nil || saves != 0 {
		t.Fatal("invalid pattern saved", err, saves)
	}
	failSave = true
	move, _ := json.Marshal(config.PriorityEdit{List: config.ListPlayback, Op: "move", Index: 1, Value: "up"})
	if err := i.applyEdit(string(move)); err == nil || !strings.Contains(err.Error(), "reload") {
		t.Fatal("stale save accepted", err)
	}
	if i.running.Studio.Playback[0].Pattern != before {
		t.Fatal("refused edit changed the running configuration")
	}
}

func TestEditGoroutineReloadsOnlyAfterSave(t *testing.T) {
	i := editInstance(t, func(string, json.RawMessage, json.RawMessage) error { return nil })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { i.runEdits(ctx); close(done) }()
	defer func() { cancel(); <-done }()
	bad, _ := json.Marshal(config.PriorityEdit{List: config.ListPlayback, Op: "remove", Index: 9})
	good, _ := json.Marshal(config.PriorityEdit{List: config.ListPlayback, Op: "remove", Index: 0})
	i.edits <- string(bad)
	i.edits <- string(good)
	if a := <-i.actions; a.Kind != control.Reload.Kind {
		t.Fatal(a)
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.editErr != "" || len(i.running.Studio.Playback) != 2 {
		t.Fatal(i.editErr, i.running.Studio.Playback)
	}
}
