package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/routing"
)

func graphFixture() screen {
	snapshot := model.Snapshot{Edition: 3, Assignments: map[string]string{"A1": "Volt", "A2": "Speakers", "input:1": "", "input:2": "", "input:3": "Webcam"}, Numbers: map[string]float32{}, Element: &model.ProcessStatus{Known: true, Running: true}}
	for strip := 0; strip < 8; strip++ {
		for _, bus := range graphBuses(3) {
			snapshot.Numbers[fmt.Sprintf("Strip[%d].%s", strip, bus)] = 0
		}
	}
	snapshot.Numbers["Patch.asio[0]"] = 1
	snapshot.Numbers["Patch.asio[1]"] = 1
	return screen{tab: 1, width: 100, height: 40, state: State{Live: true, Connected: true, Snapshot: snapshot, Intent: &config.Intent{Recording: &config.RecordingChoices{}}}}
}

func TestGraphObservedPaths(t *testing.T) {
	for _, tt := range []struct {
		name   string
		sends  map[string]float32
		want   []string
		absent string
	}{
		{"direct", map[string]float32{"Strip[0].B3": 1}, []string{"A1 ASIO 1/1 (Volt)", "[B3] --> [Discord / app mic]"}, "External host"},
		{"processed", map[string]float32{"Strip[0].B2": 1, "Strip[6].B3": 1}, []string{"[B2] --> [Element send]", "[AUX / return]", "External host path (not verified):"}, "[A2]"},
		{"pre monitor and recording", map[string]float32{"Strip[0].A2": 1, "Strip[0].B1": 1}, []string{"[A2] --> [Speakers]", "[B1] --> [Recording mix]"}, "[AUX / return]"},
		{"post monitor and recording", map[string]float32{"Strip[6].A2": 1, "Strip[6].B1": 1}, []string{"[AUX / return]", "[A2] --> [Speakers]", "[B1] --> [Recording mix]"}, "[Input 1:"},
		{"off playback retained", map[string]float32{"Strip[5].A2": 1}, []string{"[VAIO / computer]", "[A2] --> [Speakers]"}, "[Input 1:"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := graphFixture()
			for key, value := range tt.sends {
				s.state.Snapshot.Numbers[key] = value
			}
			text := strings.Join(s.graphLines(90), "\n")
			for _, want := range tt.want {
				if !strings.Contains(text, want) {
					t.Fatalf("missing %q in\n%s", want, text)
				}
			}
			if strings.Contains(text, tt.absent) {
				t.Fatalf("unexpected %q in\n%s", tt.absent, text)
			}
		})
	}
}

func TestGraphDoesNotPresentIntentAsApplied(t *testing.T) {
	s := graphFixture()
	s.state.Intent.Source = "off"
	s.state.Snapshot.Numbers["Strip[0].B3"] = 1
	s.state.Snapshot.Numbers["Strip[5].A2"] = 1
	s.state.Plan = &routing.Plan{Topology: &routing.Topology{PlaybackTarget: "A1", Operations: []routing.Operation{{Parameter: "Strip[5].A1", Value: 1, Change: true}}}}
	text := strings.Join(s.graphLines(90), "\n")
	for _, want := range []string{"Changes pending", "[B3]", "[A2] --> [Speakers]"} {
		if !strings.Contains(text, want) {
			t.Fatal(text)
		}
	}
	if strings.Contains(text, "[A1] -->") {
		t.Fatal("desired edge presented as applied")
	}
	delete(s.state.Snapshot.Numbers, "Strip[5].A2")
	s.state.Snapshot.Assignments["A2"] = ""
	s.state.Snapshot.Numbers["Strip[0].A2"] = 1
	text = strings.Join(s.graphLines(90), "\n")
	if !strings.Contains(text, "1 strip sends unknown") || !strings.Contains(text, "[no device]") {
		t.Fatal(text)
	}
	s.state.Connected = false
	if text = strings.Join(s.graphLines(90), "\n"); strings.Contains(text, "-->") || !strings.Contains(text, "unavailable") {
		t.Fatal(text)
	}
}

func TestGraphRecorderAndLegacy(t *testing.T) {
	s := graphFixture()
	s.state.Recorder = &model.RecorderSnapshot{Values: map[string]float32{}}
	for _, key := range model.RecorderParameters() {
		s.state.Recorder.Values[key] = 0
	}
	s.state.Recorder.Values["Recorder.stop"] = 1
	s.state.Recorder.Values["Recorder.B2"] = 1
	s.state.Recorder.Values["Recorder.mode.recbus"] = 1
	s.state.Recorder.Values["Recorder.ArmBus[5]"] = 1
	text := strings.Join(s.graphLines(90), "\n")
	for _, want := range []string{"[Tape: Stopped]", "[B2] --> [Element send]", "[B1]\n  `--> [Recorder: Stopped]"} {
		if !strings.Contains(text, want) {
			t.Fatal(text)
		}
	}
	s.state.Snapshot.Edition = 2
	s.state.Intent = nil
	s.state.Recorder = nil
	s.state.Snapshot.Numbers["Strip[3].B2"] = 1
	text = strings.Join(s.graphLines(90), "\n")
	if !strings.Contains(text, "[VAIO / computer]") || !strings.Contains(text, "[Virtual output 2]") || strings.Contains(text, "Element send") {
		t.Fatal(text)
	}
}

func TestGraphNavigationBoundsAndSanitization(t *testing.T) {
	s := graphFixture()
	s.actions = make(chan Action, 8)
	s.state.Snapshot.Assignments["A2"] = "speaker\x1b[2J\nINJECT" + strings.Repeat("x", 200)
	s.state.Snapshot.Numbers["Strip[5].A2"] = 1
	for _, width := range []int{20, 42, 60, 100, 160} {
		for _, height := range []int{3, 10, 30} {
			s.width, s.height = width, height
			text := s.View().Content
			if len(strings.Split(text, "\n")) > height {
				t.Fatal("height overflow")
			}
			for _, line := range strings.Split(text, "\n") {
				if ansi.StringWidth(line) > width-1 {
					t.Fatal("width overflow", width, line)
				}
			}
			if strings.Contains(text, "\x1b[2J") || strings.Contains(text, "\nINJECT") {
				t.Fatal("unsafe device name")
			}
		}
	}
	s.width, s.height = 100, 30
	s.selected = 2
	for _, key := range []tea.KeyPressMsg{{Code: tea.KeyTab}, {Code: tea.KeyTab}, {Code: tea.KeyLeft}, {Code: tea.KeyRight}} {
		previous := s.tab
		updated, _ := s.Update(key)
		s = updated.(screen)
		if s.tab == previous || s.selected != 2 {
			t.Fatal("tab navigation changed selection or did not wrap")
		}
	}
	s.tab = 1
	updated, _ := s.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	s = updated.(screen)
	if s.offset == 0 {
		t.Fatal("graph did not scroll")
	}
	s.tab = 0
	s.picker = &sourcePicker{}
	s.pickerKey("shift+tab")
	if s.tab != 1 || s.picker != nil || len(s.actions) != 0 {
		t.Fatal("picker navigation submitted action")
	}
}
