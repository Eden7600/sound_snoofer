package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"sound-snoofer/internal/config"
)

func editIntent(i *config.Intent, a Action) error {
	switch a.Row {
	case "record-mic", "record-computer", "record-tap":
		if i.Recording == nil {
			return fmt.Errorf("recording profile not configured")
		}
		if a.Row == "record-tap" {
			i.Recording.MicTap = a.Value
		} else {
			b, e := strconv.ParseBool(a.Value)
			if e != nil {
				return e
			}
			if a.Row == "record-mic" {
				i.Recording.MicEnabled = b
			} else {
				i.Recording.ComputerEnabled = b
			}
		}
	case "source":
		i.Source = a.Value
		i.Enabled = a.Value != "off"
	case "mode":
		i.Mode = a.Value
	case "monitor":
		i.Monitor = a.Value
	case "voice":
		b, e := strconv.ParseBool(a.Value)
		if e != nil {
			return e
		}
		i.Enabled = b
	default:
		if !strings.HasPrefix(a.Row, "playback:") {
			return fmt.Errorf("unknown rule")
		}
		key := strings.TrimPrefix(a.Row, "playback:")
		if _, ok := i.Playback[key]; !ok {
			return fmt.Errorf("playback rule no longer exists")
		}
		b, e := strconv.ParseBool(a.Value)
		if e != nil {
			return e
		}
		i.Playback[key] = b
	}
	i.NormalizeRecordingStage()
	return nil
}

type ruleRow struct {
	key, label, value string
	toggle            bool
}

func (s screen) rules() []ruleRow {
	i := s.state.Intent
	if i == nil {
		return nil
	}
	source := i.Source
	if !i.MicActive() {
		source = "off"
	}
	rows := []ruleRow{{"source", "Source", source, false}, {"mode", "Processing", i.Mode, false}, {"monitor", "Monitor", i.Monitor, false}}
	keys := []string{}
	for k := range i.Playback {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		rows = append(rows, ruleRow{"playback:" + k, "App playback " + k, strconv.FormatBool(i.Playback[k]), true})
	}
	if i.Recording != nil {
		r := i.Recording
		rows = append(rows,
			ruleRow{"record-computer", "Record Computer Audio", strconv.FormatBool(r.ComputerEnabled), true},
			ruleRow{"record-mic", "Record Microphone", strconv.FormatBool(r.MicEnabled), true},
			ruleRow{"record-tap", "Recording Mic Stage", r.MicTap, false},
			ruleRow{"record-start", "Start Recording", "Enter (live only)", false},
			ruleRow{"record-stop", "Stop Recording", "Enter (live only)", false})
	}
	return rows
}
func cycle(current string, values ...string) string {
	for n, v := range values {
		if v == current {
			return values[(n+1)%len(values)]
		}
	}
	return values[0]
}
func (s *screen) ruleAction(key string) {
	rows := s.rules()
	if len(rows) == 0 {
		return
	}
	row := rows[min(s.selected, len(rows)-1)]
	if row.key == "source" && (key == "enter" || key == " ") {
		s.openSource()
		return
	}
	if key == "enter" && (row.key == "record-start" || row.key == "record-stop") {
		kind := startRecording
		if row.key == "record-stop" {
			kind = stopRecording
		}
		s.queue(Action{Kind: kind, Revision: s.state.Revision})
		return
	}
	if row.key == "record-start" || row.key == "record-stop" {
		return
	}
	if key != " " && key != "enter" {
		return
	}
	value := row.value
	switch row.key {
	case "record-tap":
		if s.state.Intent.Mode == "direct" {
			s.pending = "Pre only in Direct"
			return
		}
		value = cycle(value, "pre", "post")
	case "source":
		value = cycle(value, "desk", "lav", "webcam", "off")
	case "mode":
		value = cycle(value, "direct", "element")
	case "monitor":
		value = cycle(value, "off", "pre", "post")
	default:
		value = strconv.FormatBool(value != "true")
	}
	s.queue(Action{Kind: editRule, Row: row.key, Value: value, Revision: s.state.Revision})
}
func (s *screen) queue(a Action) {
	select {
	case s.actions <- a:
		s.pending = "Queued"
	default:
		s.pending = "A command is already queued"
	}
}
func (s screen) ruleLines() []string {
	rows := []string{"ROUTING RULES  (choices persist even in dry-run)"}
	if s.state.Intent == nil {
		return append(rows, "No voice profile configured. Add studio.voice to enable these controls.")
	}
	status := "Pending"
	if !s.state.Live {
		status = "Preview (no mixer writes)"
	} else if s.state.Plan != nil && !s.state.Plan.HasChanges() {
		status = "Applied (mixer readback)"
	}
	if s.state.Error != "" || s.state.StateError != "" {
		status = "Error"
	} else if s.state.Plan != nil && s.state.Plan.HasUnresolved() {
		status = "Inactive routes: see Routing for unavailable destinations"
	}
	rows = append(rows, "State: "+status)
	rows = append(rows, "r: reload  f: refresh  x: reset saved choices")
	for n, row := range s.rules() {
		mark := "  "
		if n == s.selected {
			mark = "> "
		}
		value := row.value
		if row.toggle {
			value = "Enabled"
			if row.value == "false" {
				value = "Disabled"
			}
			if row.key == "voice" {
				value = "On"
				if row.value == "false" {
					value = "Off (all mic sends disconnected when applied)"
				}
			}
		}
		rows = append(rows, fmt.Sprintf("%s%-27s %s", mark, row.label, value))
	}
	if s.state.Plan != nil && s.state.Plan.Topology != nil && s.state.Plan.Topology.Voice != nil {
		v := s.state.Plan.Topology.Voice
		rows = append(rows, "", "Effective mic: "+v.Effective, "Preferred mic: "+v.Preferred, "Monitoring: "+v.Monitor)
		if v.Reason != "" {
			rows = append(rows, v.Reason)
		}
	}
	if s.state.Intent.Recording != nil {
		rows = append(rows, "", "RECORDER: "+s.state.Recorder.State(), s.state.RecorderNotice, "Recording continues in Voicemeeter when this TUI exits or goes dry.")
		if s.state.Plan != nil && s.state.Plan.Topology != nil && s.state.Plan.Topology.Recording != nil {
			r := s.state.Plan.Topology.Recording
			rows = append(rows, "Recording mic: "+r.Mic, "Computer capture: "+strings.Join(r.ComputerSources, ", "))
			if r.Blocked != "" {
				rows = append(rows, r.Blocked)
			}
		}
		if s.state.Recorder != nil && s.state.Recorder.Values["Bus[5].Mute"] == 1 {
			rows = append(rows, "WARNING: B1 is muted")
		}
	}
	rows = append(rows, "", "LOCKED: AUX -> B2 OFF (feedback protection)", "RESERVED: B2 = Element send; AUX = return; B3 = app microphone", "Element audio is not verified by mixer readback. Use Direct for recovery.", "Pre monitoring is before Element; monitoring speakers can feed back.", "", "Observed voice sends:")
	for _, strip := range []int{0, 1, 2, 6} {
		b2, ok2 := s.state.Snapshot.Numbers[fmt.Sprintf("Strip[%d].B2", strip)]
		b3, ok3 := s.state.Snapshot.Numbers[fmt.Sprintf("Strip[%d].B3", strip)]
		if ok2 && ok3 {
			rows = append(rows, fmt.Sprintf("Strip %d: B2=%g  B3=%g", strip+1, b2, b3))
		} else {
			rows = append(rows, fmt.Sprintf("Strip %d: observation unavailable", strip+1))
		}
	}
	if s.state.StateError != "" {
		rows = append(rows, s.state.StateError, "x: Reset saved choices to configured defaults")
	}
	return rows
}
