package tui

import (
	"slices"
	"strings"
)

type sourcePicker struct {
	row, title string
	options    []string
	selected   int
	current    string
	revision   uint64
}

func (s *screen) openSource() {
	s.openChoice("source")
}

func (s screen) choiceValue(row string) string {
	if s.choices() == nil {
		return ""
	}
	if row == "output" {
		return s.choices().PlaybackDevice
	}
	return s.choices().Source
}

func (s screen) choiceOptions(row string) []string {
	if row == "output" {
		if len(s.state.OutputOptions) > 0 {
			return s.state.OutputOptions
		}
		return []string{""}
	}
	if len(s.state.MicOptions) > 0 {
		return s.state.MicOptions
	}
	return []string{"off"}
}

func (s *screen) openChoice(row string) {
	title := "Source"
	if row == "output" {
		title = "Playback Device"
	}
	s.picker = &sourcePicker{row: row, title: title, current: s.choiceValue(row), revision: s.state.Revision}
	s.refreshPicker(s.picker.current)
}

func (s *screen) refreshPicker(highlight string) {
	s.picker.options = append([]string(nil), s.choiceOptions(s.picker.row)...)
	s.picker.selected = slices.Index(s.picker.options, highlight)
	if s.picker.selected < 0 {
		fallback := "off"
		if s.picker.row == "output" {
			fallback = ""
		}
		s.picker.selected = max(0, slices.Index(s.picker.options, fallback))
	}
}

// pickerKey consumes all input except quit, so edits never leak behind the list.
func (s *screen) pickerKey(key string) {
	switch key {
	case "down", "j":
		s.picker.selected = min(s.picker.selected+1, len(s.picker.options)-1)
	case "up", "k":
		s.picker.selected = max(0, s.picker.selected-1)
	case "esc":
		s.picker = nil
	case "tab", "shift+tab":
		s.picker = nil
		step := 1
		if key == "shift+tab" {
			step = 3
		}
		s.tab = (s.tab + step) % 4
		s.offset = 0
	case "enter":
		choice := s.picker.options[s.picker.selected]
		if !slices.Contains(s.choiceOptions(s.picker.row), choice) {
			s.refreshPicker(choice)
			s.pending = "Device disconnected; choose again"
			return
		}
		if choice != s.picker.current {
			s.queue(Action{Kind: editRule, Row: s.picker.row, Value: choice, Revision: s.picker.revision})
		}
		s.picker = nil
	}
}

// Place the bounded list beneath Source when it fits, otherwise center it.
func (s screen) pickerOverlay(rows []string, width, anchor int) []string {
	if s.picker == nil || len(rows) < 6 || width < 40 {
		return rows
	}
	boxWidth := min(60, width-4)
	box := []string{paint("38;5;110", "╭─ "+s.picker.title+" "+strings.Repeat("─", boxWidth-len(s.picker.title)-4)+"╮")}
	visible := min(len(s.picker.options), len(rows)-4)
	start := max(0, s.picker.selected-visible+1)
	for n := start; n < start+visible; n++ {
		source := s.picker.options[n]
		mark := "  "
		if n == s.picker.selected {
			mark = "› "
		}
		label := pretty(source)
		if source == s.picker.current {
			label += "  ✓"
		}
		line := "│ " + fit(mark+label, boxWidth-4) + " │"
		if n == s.picker.selected {
			line = paint("1;38;5;231;48;5;24", line)
		} else {
			line = paint("38;5;252;48;5;235", line)
		}
		box = append(box, line)
	}
	box = append(box, paint("38;5;110", "╰"+strings.Repeat("─", boxWidth-2)+"╯"))
	if anchor+len(box) > len(rows)-1 {
		anchor = max(1, (len(rows)-len(box))/2)
	}
	for n, line := range box {
		// Replace whole visual rows to avoid cutting trusted ANSI sequences.
		rows[anchor+n] = "  " + line
	}
	return rows
}
