package app

import (
	"fmt"
	"os"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"sound-snoofer/snoofer"
)

// Keep styling local to the surface; plugin strings never contain trusted ANSI.
func paint(code, text string) string {
	if os.Getenv("NO_COLOR") != "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}
func fit(text string, width int) string {
	text = ansi.Truncate(clean(text), max(0, width), "…")
	return text + strings.Repeat(" ", max(0, width-ansi.StringWidth(text)))
}
func controlValue(c snoofer.Control) string {
	value := c.Value
	if label, ok := c.OptionLabels[value]; ok {
		value = label
	}
	if c.Kind == "toggle" {
		if c.Value == "On" {
			value = "● On"
		} else if c.Value == "Off" {
			value = "○ Off"
		}
	}
	if c.Kind == "selection" {
		if value == "" {
			value = "Automatic / empty"
		}
		value += " ▾"
	}
	if c.Kind == "command" {
		value = "Enter ↵"
	}
	if !c.Available {
		value = "Unavailable"
	}
	return value
}
func row(label, value string, width int, selected, subdued bool) string {
	prefix := "  "
	if selected {
		prefix = "› "
	}
	labelWidth := min(32, max(12, width/2))
	if value == "" {
		labelWidth = width - 2
	}
	text := prefix + fit(label, labelWidth)
	if value != "" {
		text += " " + fit(value, max(0, width-labelWidth-3))
	}
	text = fit(text, width)
	color := "38;5;252"
	if subdued {
		color = "38;5;245"
	}
	if selected {
		color = "48;5;24;38;5;255"
		if subdued {
			color = "48;5;238;38;5;250"
		}
	}
	return paint(color, text)
}
func (s screen) View() tea.View {
	width, height := max(1, min(112, s.width-1)), max(1, s.height)
	finish := func(lines []string) tea.View {
		if len(lines) > height {
			lines = lines[:height]
		}
		for n, line := range lines {
			lines[n] = ansi.Truncate(line, width, "…")
		}
		v := tea.NewView(strings.Join(lines, "\n"))
		v.AltScreen = true
		return v
	}
	if s.width < 42 || s.height < 10 {
		return finish([]string{paint("1;38;5;110", " SNOOFER"), " Enlarge terminal to at least 42 × 10.", " Editing paused · q closes controls"})
	}
	running := 0
	for _, status := range s.state.Plugins {
		if status == "Running" {
			running++
		}
	}
	title := paint("1;38;5;255", " SNOOFER") + paint("38;5;245", fmt.Sprintf("  /  %d plugins running", running))
	tabs := []string{}
	for n, label := range []string{"Controls", "Plugins"} {
		code := "38;5;245"
		text := "  " + label + "  "
		if n == s.tab {
			code = "1;48;5;24;38;5;255"
			text = "  [" + label + "]  "
		}
		tabs = append(tabs, paint(code, text))
	}
	inner := width - 4
	rail := 0
	if width >= 79 && s.tab == 0 && !s.editing && len(s.groups()) > 0 {
		rail = 25
		inner -= rail
	}
	body := []string{}
	selectedLine := 0
	section := func(name string) {
		if len(body) > 0 {
			body = append(body, "")
		}
		body = append(body, paint("1;38;5;110", fit("  "+strings.ToUpper(clean(name)), inner)))
	}
	detail := "Ready"
	footer := "←→ section · ↑↓ select · Enter edit · +/- gain · Tab plugins · r retry · q close"
	if width < 79 {
		footer = "←→ section · Enter edit · Tab plugins · q close"
	}
	if s.editing {
		section("Edit " + s.target.Label)
		detail = "Enter applies"
		footer = "↑↓ choose · Enter apply · Esc cancel"
		if len(s.options) > 0 {
			for n, value := range s.options {
				if label, ok := s.target.OptionLabels[value]; ok {
					value = label
				}
				if value == "" {
					value = "Automatic / empty"
				}
				if n == s.option {
					selectedLine = len(body)
				}
				body = append(body, row(value, "", inner, n == s.option, false))
			}
		} else {
			selectedLine = len(body)
			body = append(body, row(s.text+"▏", "", inner, true, false))
			detail = "Enter value"
			footer = "Enter apply · Esc cancel"
		}
	} else if s.tab == 1 {
		section("Plugin management")
		ids := s.plugins()
		if len(ids) == 0 {
			body = append(body, fit("  No plugins compiled into this build.", inner))
		}
		for n, id := range ids {
			value := "○ Disabled"
			if s.state.Enabled[id] {
				value = "● " + s.state.Plugins[id]
			}
			if n == s.selected {
				selectedLine = len(body)
				detail = id + " · " + s.state.Plugins[id] + ""
			}
			body = append(body, row(id, value, inner, n == s.selected, !s.state.Enabled[id]))
		}
		footer = "↑↓ select · Enter enable/disable · Tab controls · r retry · q close"
	} else {
		controls := s.rows()
		if len(controls) > 0 {
			name := s.currentGroup()
			if rail == 0 {
				name = fmt.Sprintf("%s  %d/%d", name, slices.Index(s.groups(), name)+1, len(s.groups()))
			}
			section(name)
		}
		if len(controls) == 0 {
			section("Welcome")
			body = append(body, fit("  Enable plugins in the Plugins tab.", inner))
		}
		for n, c := range controls {
			if n == s.selected {
				selectedLine = len(body)
				detail = c.Label
				if c.Status != "" {
					detail += " · " + c.Status
				}
				if !c.Available {
					detail += " · Unavailable"
				}
			}
			body = append(body, row(c.Label, controlValue(c), inner, n == s.selected, c.Subdued))
		}
	}
	contentHeight := len(body)
	if rail > 0 {
		contentHeight = max(contentHeight, len(s.groups()))
	}
	capacity := min(height-7, max(6, contentHeight))
	start := max(0, selectedLine-capacity+1)
	start = min(start, max(0, len(body)-capacity))
	visible := body[start:min(len(body), start+capacity)]
	edge := paint("38;5;239", "│")
	lines := []string{title, strings.Join(tabs, " "), "", paint("38;5;239", "╭"+strings.Repeat("─", width-2)+"╮")}
	for n := 0; n < capacity; n++ {
		text := ""
		if n < len(visible) {
			text = visible[n]
		}
		text = ansi.Truncate(text, inner, "…")
		text += strings.Repeat(" ", max(0, inner-ansi.StringWidth(text)))
		if rail > 0 {
			groups := s.groups()
			index := slices.Index(groups, s.currentGroup())
			first := max(0, index-capacity+1)
			left := strings.Repeat(" ", rail-2)
			if n+first < len(groups) {
				name := groups[n+first]
				if name == "" {
					name = "General"
				}
				left = row(name, "", rail-2, n+first == index, false)
			}
			text = left + edge + " " + text
		}
		lines = append(lines, edge+" "+text+" "+edge)
	}
	scroll := ""
	if len(body) > capacity {
		scroll = fmt.Sprintf(" %d–%d / %d ", start+1, min(len(body), start+capacity), len(body))
	}
	lines = append(lines, paint("38;5;239", "╰"+scroll+strings.Repeat("─", max(0, width-2-ansi.StringWidth(scroll)))+"╯"))
	notice := s.state.Notice
	if s.notice != "" {
		notice = s.notice
	}
	if notice != "" {
		detail = notice
	}
	if s.state.Confirmation != "" {
		detail = s.state.Confirmation
		footer = "y confirm and restart · n / Esc cancel"
	}
	detailColor := "38;5;252"
	if s.state.Confirmation != "" {
		detailColor = "38;5;179"
	}
	lines = append(lines, paint(detailColor, fit(" "+detail, width)), paint("38;5;245", fit(" "+footer, width)))
	return finish(lines)
}
