package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func paint(code, text string) string {
	if os.Getenv("NO_COLOR") != "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}
func fit(text string, w int) string {
	text = ansi.Truncate(clean(text), max(0, w), "…")
	return text + strings.Repeat(" ", max(0, w-ansi.StringWidth(text)))
}
func pretty(v string) string {
	switch v {
	case "":
		return "Automatic"
	case "desk":
		return "Desk mic · Volt 1"
	case "lav":
		return "Lav mic · Volt 2"
	case "webcam":
		return "Webcam"
	case "off":
		return "Off"
	case "element":
		return "Element / VST"
	case "direct":
		return "Direct"
	case "pre":
		return "Pre"
	case "post":
		return "Post"
	}
	return v
}
func group(r ruleRow) string {
	if r.key == "record-start" || r.key == "record-stop" || strings.HasPrefix(r.key, "snippet-") {
		return "ACTIONS"
	}
	if strings.HasPrefix(r.key, "record-") {
		return "RECORDING"
	}
	if r.key == "output" || strings.HasPrefix(r.key, "playback:") {
		return "COMPUTER AUDIO"
	}
	return "MICROPHONE"
}
func (s screen) dashboardRows(w int) ([]string, int) {
	rows := []string{}
	selectedLine := 0
	last := ""
	for n, r := range s.rules() {
		g := group(r)
		if g != last {
			if last != "" {
				rows = append(rows, strings.Repeat(" ", w))
			}
			rows = append(rows, paint("38;5;110", fit("  "+g, w)))
			last = g
		}
		value := pretty(r.value)
		if r.key == "monitor" && (r.value == "pre" || r.value == "post") {
			value += "-VST"
		}
		if r.key == "source" || r.key == "output" {
			value += " ▾"
		}
		if r.toggle {
			value = "○ Off"
			if r.value == "true" {
				value = "● On"
			}
		}
		if r.key == "record-start" {
			value = "↵ Start"
		}
		if r.key == "snippet-play" {
			value = "↵ Play"
		}
		if r.key == "record-stop" || r.key == "snippet-stop" {
			value = "↵ Stop"
		}
		statusColor, statusText := s.controlStatus(r)
		if statusColor == "196" {
			value = statusText
		} else if statusText != "" {
			value += " · " + statusText
		}
		label := r.label
		if statusColor != "" && w < 60 {
			switch r.key {
			case "record-tap":
				label = "Mic Stage"
			case "record-computer":
				label = "Record Computer"
			case "record-vst":
				label = "Tape to VST"
			case "output":
				label = "Output"
			}
		}
		if strings.HasPrefix(r.key, "playback:") {
			label = "Playback " + strings.TrimPrefix(r.key, "playback:")
		}
		prefix := "  "
		if n == s.selected {
			prefix = "› "
			selectedLine = len(rows)
		}
		text := prefix + label
		gap := w - ansi.StringWidth(text) - ansi.StringWidth(value) - 2
		if gap < 2 {
			gap = 2
		}
		line := fit(text+strings.Repeat(" ", gap)+value, w)
		if n == s.selected {
			color := "231"
			if statusColor != "" {
				color = statusColor
			}
			line = paint("1;38;5;"+color+";48;5;24", line)
		} else {
			color := "252"
			if statusColor != "" {
				color = statusColor
			}
			line = paint("38;5;"+color, line)
		}
		rows = append(rows, line)
	}
	if len(rows) == 0 {
		rows = []string{fit("No voice profile configured", w)}
	}
	return rows, selectedLine
}
func (s screen) micStatus() string {
	if s.state.Intent == nil {
		return "Not configured"
	}
	if s.state.Intent.MicActive() {
		if s.state.Plan != nil && s.state.Plan.Topology != nil && s.state.Plan.Topology.Voice != nil {
			return pretty(s.state.Plan.Topology.Voice.Effective)
		}
		return "Awaiting observation"
	}
	if !s.state.Live {
		return "Off selected · preview only"
	}
	if !s.state.Connected || s.state.Error != "" {
		return "Off pending · observation unavailable"
	}
	for _, n := range []int{0, 1, 2, 6} {
		for _, letter := range []string{"A", "B"} {
			count := 5
			if letter == "B" {
				count = 3
			}
			for b := 1; b <= count; b++ {
				v, ok := s.state.Snapshot.Numbers[fmt.Sprintf("Strip[%d].%s%d", n, letter, b)]
				if !ok || v != 0 {
					return "Off pending · sends remain"
				}
			}
		}
	}
	return "Off · sends disconnected"
}
func (s screen) statusRows(w int) []string {
	rows := []string{"SIGNAL STATUS", "", "Microphone", s.micStatus(), "", "Playback output"}
	output := "Awaiting devices"
	if s.state.Plan != nil && s.state.Plan.Topology != nil {
		t := s.state.Plan.Topology
		if t.PlaybackTarget != "" {
			output = t.PlaybackTarget + " · " + empty(s.state.Snapshot.Assignments[t.PlaybackTarget])
		}
	}
	rows = append(rows, output, "", s.state.Snapshot.ElementStatus())
	if s.state.Plan != nil && s.state.Plan.Topology != nil && s.state.Plan.Topology.Voice != nil {
		v := s.state.Plan.Topology.Voice
		rows = append(rows, "Processing: "+pretty(v.EffectiveMode))
		if v.ProcessingReason != "" {
			rows = append(rows, v.ProcessingReason)
		}
	}
	rows = append(rows, "", "Native recorder", s.state.Recorder.State())
	if s.state.Intent != nil && s.state.Intent.Recording != nil {
		if s.state.Plan != nil && s.state.Plan.Topology != nil && s.state.Plan.Topology.Recording != nil {
			r := s.state.Plan.Topology.Recording
			rows = append(rows, "Mic: "+r.Mic, fmt.Sprintf("Eligible capture sources: %d", r.Sources))
			if r.Blocked != "" {
				rows = append(rows, r.Blocked)
			}
		}
	}
	rows = append(rows, "", "QUICK HELP", "L   Toggle live / preview", "R   Reload saved choices", "X   Reset saved choices", "", "B1  Recording mix", "B2  Element send", "B3  Discord / app microphone")
	out := make([]string, len(rows))
	for n, line := range rows {
		code := "38;5;246"
		if n == 0 || line == "QUICK HELP" {
			code = "1;38;5;110"
		}
		out[n] = paint(code, fit(" "+line, w))
	}
	return out
}

func (s screen) dashboardView() tea.View {
	width, height := s.width, s.height
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 30
	}
	w := max(1, width-1)
	mode := "PREVIEW · no mixer writes"
	modeColor := "38;5;221"
	if s.state.Live {
		mode = "LIVE"
		modeColor = "38;5;120"
	}
	connection := "Connecting"
	if s.state.Connected {
		connection = "Connected"
	}
	if s.state.Error != "" {
		connection = "Needs attention"
	}
	out := []string{paint(modeColor, fit("  "+mode+"   ·   "+connection+"   ·   Recorder: "+s.state.Recorder.State(), w))}
	tabs := []string{"Controls", "Graph"}
	for n := range tabs {
		if n == s.tab {
			tabs[n] = "[ " + tabs[n] + " ]"
		} else {
			tabs[n] = "  " + tabs[n] + "  "
		}
	}
	out = append(out, paint("38;5;183", fit("  "+strings.Join(tabs, "  "), w)))
	now := s.now
	if now.IsZero() {
		now = time.Now()
	}
	notice := s.state.noticeAt(now)
	if s.pending != "" && s.state.Error == "" && s.state.StateError == "" && (s.state.NoticeKind != noticeError || notice == "") {
		notice = s.pending
	}
	if notice != "" {
		out = append(out, paint("38;5;246", fit("  "+notice, w)))
	}
	bodyHeight := max(1, height-len(out)-3)
	leftWidth := max(1, w-4)
	wide := s.tab == 0 && w >= 100
	if wide {
		leftWidth = (w - 7) * 3 / 5
	}
	rightWidth := w - leftWidth - 7
	title := " CONTROLS "
	if s.tab != 0 {
		title = " GRAPH "
	}
	title = clean(title)
	out = append(out, paint("38;5;60", "╭─"+title+strings.Repeat("─", max(0, w-3-ansi.StringWidth(title)))+"╮"))
	var rows []string
	selectedLine := 0
	if s.tab == 0 {
		rows, selectedLine = s.dashboardRows(leftWidth)
	} else {
		for _, line := range s.graphLines(leftWidth - 1) {
			rows = append(rows, paint("38;5;252", fit(" "+line, leftWidth)))
		}
	}
	start := min(s.offset, max(0, len(rows)-bodyHeight))
	if s.tab == 0 {
		if selectedLine < start {
			start = selectedLine
		}
		if selectedLine >= start+bodyHeight {
			start = selectedLine - bodyHeight + 1
		}
	}
	side := s.statusRows(max(1, rightWidth))
	for n := 0; n < bodyHeight; n++ {
		line := strings.Repeat(" ", leftWidth)
		if start+n < len(rows) {
			line = rows[start+n]
		}
		row := paint("38;5;60", "│ ") + line
		if wide {
			right := strings.Repeat(" ", rightWidth)
			if n < len(side) {
				right = side[n]
			}
			row += paint("38;5;60", " │ ") + right
		}
		row += " " + paint("38;5;60", "│")
		out = append(out, ansi.Truncate(row, w, ""))
	}
	out = append(out, paint("38;5;60", "╰"+strings.Repeat("─", max(0, w-2))+"╯"))
	help := "Tab views · ↑↓ select · Enter/Space change · L live · Q quit"
	if s.tab != 0 {
		help = "Tab views · ↑↓ / PgUp/PgDn scroll · R reload · F refresh · L live · Q quit"
	}
	if s.picker != nil {
		help = "↑↓ choose · Enter confirm · Esc cancel · Tab views · Q quit"
	}
	out = append(out, paint("38;5;110", fit(" "+help, w)))
	if height >= 10 && width >= 42 {
		out = s.pickerOverlay(out, w, len(out)-bodyHeight-2+selectedLine-start+1)
	}
	if height < 10 || width < 42 {
		out = []string{fit("SOUND SNOOFER", w), fit("Terminal too small; enlarge to 42×10.", w), fit("L live · Q quit", w)}
	}
	if len(out) > height {
		out = out[:height]
	}
	content := strings.Join(out, "\n")
	if s.attached {
		content = strings.ReplaceAll(content, "Q quit", "Q close")
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.WindowTitle = "Sound Snoofer"
	return v
}
