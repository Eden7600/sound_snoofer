package app

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"sound-snoofer/snoofer"
)

func styledFixture() screen {
	return screen{ctx: context.Background(), width: 100, height: 30, actions: make(chan UIAction, 8), state: ViewState{
		Plugins: map[string]string{"audio": "Running", "streamdeck": "Running", "vr": "Disabled"}, Enabled: map[string]bool{"audio": true, "streamdeck": true},
		Controls: []snoofer.Control{
			{ID: "audio.normal-source", Group: "Normal microphone", Label: "Microphone", Value: "lav", Kind: "selection", Available: true, Subdued: true, Status: "VR overriding — edits apply outside VR", OptionLabels: map[string]string{"lav": "Lavalier · Volt 2"}, Options: []string{"auto", "lav", "off"}, Operations: []string{"set"}, Revision: 1},
			{ID: "audio.mode", Group: "Normal microphone", Label: "Processing", Value: "direct", Kind: "selection", Available: true},
			{ID: "audio.playback", Group: "Normal playback", Label: "Output", Value: "SteelSeries Arena", Kind: "selection", Available: true},
			{ID: "audio.mute", Group: "Shared audio", Label: "Microphone mute", Value: "On", Kind: "toggle", Available: true},
			{ID: "audio.gain", Group: "Shared audio", Label: "Playback gain", Value: "-6.0 dB", Kind: "numeric", Available: true},
			{ID: "audio.health", Group: "System", Label: "Audio health", Value: "Processing", Kind: "status", Available: true},
		},
	}}
}
func TestStyledViewsStayInsideTerminal(t *testing.T) {
	for _, plain := range []bool{false, true} {
		t.Run(fmt.Sprint(plain), func(t *testing.T) {
			if plain {
				t.Setenv("NO_COLOR", "1")
			} else {
				t.Setenv("NO_COLOR", "")
			}
			for _, size := range [][2]int{{1, 1}, {20, 3}, {41, 9}, {42, 10}, {60, 15}, {100, 30}, {140, 40}} {
				for _, mode := range []string{"controls", "plugins", "picker", "text", "confirmation"} {
					s := styledFixture()
					s.width, s.height = size[0], size[1]
					switch mode {
					case "plugins":
						s.tab = 1
						s.selected = 2
					case "picker":
						s.editing = true
						s.target = s.state.Controls[0]
						for n := 0; n < 70; n++ {
							s.options = append(s.options, fmt.Sprintf("Choice %d", n))
						}
						s.option = 69
					case "text":
						s.editing = true
						s.target = s.state.Controls[0]
						s.text = strings.Repeat("長", 120)
					case "confirmation":
						s.state.Confirmation = "Disable audio and VR, then restart Snoofer?"
					}
					view := s.View().Content
					lines := strings.Split(view, "\n")
					if len(lines) > s.height {
						t.Fatalf("%v %s: height %d", size, mode, len(lines))
					}
					for _, line := range lines {
						if ansi.StringWidth(line) > max(1, s.width-1) {
							t.Fatalf("%v %s: overflow %q", size, mode, line)
						}
					}
					if plain && strings.Contains(view, "\x1b[") {
						t.Fatal("NO_COLOR ignored")
					}
					if s.width >= 42 && s.height >= 10 && mode == "picker" && !strings.Contains(view, "Choice 69") {
						t.Fatal("selected option scrolled away")
					}
				}
			}
		})
	}
}
func TestStyledSelectionAndResizeSafety(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	s := styledFixture()
	view := s.View().Content
	for _, want := range []string{"╭", "╯", "[Controls]", "NORMAL MICROPHONE", "›", "VR overriding", "\x1b[48;5;238"} {
		if !strings.Contains(view, want) {
			t.Fatal("missing presentation", want)
		}
	}
	s.state.Controls[0].Label = "Unsafe\x1b[2J\nlabel"
	if strings.Contains(s.View().Content, "\x1b[2J") {
		t.Fatal("plugin injected terminal commands")
	}
	s.editing = true
	s.target = s.state.Controls[0]
	s.text = "off"
	s.width = 20
	s.height = 3
	m, _ := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	s = m.(screen)
	if len(s.actions) != 0 {
		t.Fatal("hidden edit sent")
	}
	s.state.Confirmation = "Restart?"
	s.Update(tea.KeyPressMsg{Code: 'y'})
	if len(s.actions) != 0 {
		t.Fatal("hidden confirmation sent")
	}
}
func TestStyledPreview(t *testing.T) {
	t.Log("\n" + ansi.Strip(styledFixture().View().Content))
}
