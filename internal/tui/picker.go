package tui

import "strings"

var sourceOptions = []string{"desk", "lav", "webcam", "off"}

type sourcePicker struct {
	selected int
	current  string
	revision uint64
}

func (s *screen) openSource() {
	current := s.rules()[s.selected].value
	s.picker = &sourcePicker{current: current, revision: s.state.Revision}
	for n, source := range sourceOptions {
		if source == current {
			s.picker.selected = n
		}
	}
}

// pickerKey consumes all input except quit, so edits never leak behind the list.
func (s *screen) pickerKey(key string) {
	switch key {
	case "down", "j":
		s.picker.selected = min(s.picker.selected+1, len(sourceOptions)-1)
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
		choice := sourceOptions[s.picker.selected]
		if choice != s.picker.current {
			s.queue(Action{Kind: editRule, Row: "source", Value: choice, Revision: s.picker.revision})
		}
		s.picker = nil
	}
}

// Place the bounded list beneath Source when it fits, otherwise center it.
func (s screen) pickerOverlay(rows []string, width, anchor int) []string {
	if s.picker == nil || len(rows) < 9 || width < 40 {
		return rows
	}
	boxWidth := min(32, width-4)
	box := []string{paint("38;5;110", "╭─ Source "+strings.Repeat("─", boxWidth-10)+"╮")}
	for n, source := range sourceOptions {
		mark := "  "
		if n == s.picker.selected {
			mark = "› "
		}
		label := strings.ToUpper(source[:1]) + source[1:]
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
