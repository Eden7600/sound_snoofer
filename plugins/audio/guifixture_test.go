package audio

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/controller"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
	"sound-snoofer/internal/windowsaudio"
	"sound-snoofer/snoofer"
)

// guiFixtureAt fixes every timestamp the published controls carry.
var guiFixtureAt = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

// guiState is a fixed studio: the Volt interface with the lav in use, AirPods
// playback, and the built-in configuration. It never reads installed devices,
// saved choices or personal settings.
func guiState(t *testing.T, live bool) (control.State, *Instance) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "gui-config.json"))
	if err != nil {
		t.Fatal(err)
	}
	settings := Settings{Config: raw, StatePath: "audio"}
	i := &Instance{settings: settings, raw: snoofer.MarshalSettings(settings), saveSettings: func(string, json.RawMessage, json.RawMessage) error { return nil }}
	cfg, err := config.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	i.running = cfg
	s := priorityState()
	s.Intent = cfg.VoiceIntent()
	s.Intent.PauseSends = true
	s.Intent.Recording.MicEnabled = true
	s.SilentMics = []string{"desk"}
	recorder := model.RecorderSnapshot{Values: map[string]float32{}}
	for _, p := range model.RecorderParameters() {
		recorder.Values[p] = 0
	}
	recorder.Values["Recorder.play"] = 1
	s.Recorder = &recorder
	s.ActiveIntent = s.Intent
	s.Live, s.Connected = live, true
	s.Health, s.Profile = "Healthy", "Normal"
	s.ObservedAt, s.LevelsAt = guiFixtureAt, guiFixtureAt
	s.ChoiceLabels = map[string]string{}
	for _, m := range cfg.Studio.Mics() {
		s.ChoiceLabels[m.ID] = m.Name
	}
	s.OutputOptions = []string{"", "Headphones (AirPods Pro)", "Speakers (SteelSeries Arena 7)", "Game (SteelSeries Arena 7)"}
	s.Intent.PlaybackDevice = "Headphones (AirPods Pro)"
	s.Plan.Topology.Voice = &routing.VoiceStatus{Effective: "lav", EffectiveMode: "direct"}
	s.Intent.Monitor = "pre"
	s.Plan.Topology.HeldSends = 3
	// Identity suggestions, an ambiguous Arena pattern, a second interface and an unplugged webcam.
	for n, d := range s.Snapshot.Devices {
		switch d.Name {
		case "Focusrite USB ASIO":
			s.Snapshot.Devices[n].ID = "{focusrite}"
		case "Microphone (Insta360 Link 2)":
			s.Snapshot.Devices[n].Available = false
		}
	}
	// Windows endpoint IDs give WDM suggestions and the AirPods entry their identity.
	s.DefaultsDetail.Endpoints = []windowsaudio.Endpoint{
		{ID: "{airpods}", Name: "Headphones (AirPods Pro)", Flow: 0},
		{ID: "{realtek}", Name: "Speakers (Realtek Audio)", Flow: 0},
		{ID: "{realtek-mic}", Name: "Microphone (Realtek Audio)", Flow: 1},
	}
	s.MicOptions = []string{"desk", "lav", "off"}
	s.Snapshot.Devices = append(s.Snapshot.Devices,
		model.Device{Name: "Headphones (SteelSeries Arena 7)", Direction: "output", Driver: "wdm", Available: true},
		model.Device{Name: "Analogue 1 + 2 (Focusrite USB)", Direction: "input", Driver: "wdm", Available: true})
	s.Plan.Topology.Outputs = []routing.OutputStatus{{ID: "music", Name: "Music", Device: "Speakers (SteelSeries Arena 7)", Bus: "A3", State: routing.OutputOK, Sources: []string{"virtual:1", "tape"}}}
	s.Snapshot.Numbers = map[string]float32{}
	s.Levels = map[string]float32{}
	for _, target := range []string{"playback", "mic"} {
		parameter := controller.GainTarget(s.Plan, target)
		s.Snapshot.Numbers[parameter] = -6
		s.Levels[parameter] = 0.178 // -15 dBFS
	}
	return s, i
}

// TestGUIFixture keeps app/web/test/fixtures/audio.*.json equal to what the
// audio provider publishes for guiState. SNOOFER_UPDATE_FIXTURES=1 rewrites them.
func TestGUIFixture(t *testing.T) {
	for _, variant := range []struct {
		name string
		live bool
	}{{"live", true}, {"preview", false}} {
		t.Run(variant.name, func(t *testing.T) {
			s, i := guiState(t, variant.live)
			published := append(controls(s), slotControls(s)...)
			published = append(published, i.priorityControls(s)...)
			registry := snoofer.NewControls()
			if err := registry.Publish("audio", published, nil); err != nil {
				t.Fatal(err)
			}
			got, err := json.MarshalIndent(registry.Snapshot(), "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, '\n')
			path := filepath.Join("..", "..", "app", "web", "test", "fixtures", "audio."+variant.name+".json")
			if os.Getenv("SNOOFER_UPDATE_FIXTURES") == "1" {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err, "; run with SNOOFER_UPDATE_FIXTURES=1 to create it")
			}
			if !bytes.Equal(bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n")), got) {
				t.Fatalf("%s is out of date with the audio provider; run with SNOOFER_UPDATE_FIXTURES=1 and review the diff", path)
			}
		})
	}
}
