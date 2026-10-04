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
	case "record-vst", "record-loop":
		if i.Recording == nil {
			return fmt.Errorf("recording profile not configured")
		}
		value, err := strconv.ParseBool(a.Value)
		if err != nil {
			return err
		}
		if a.Row == "record-vst" {
			if value && (!i.MicActive() || i.Mode != "element") {
				return fmt.Errorf("Recording to VST requires an active source and Element mode")
			}
			i.Recording.ToVST = value
		} else {
			i.Recording.Loop = value
		}
	case "output":
		i.PlaybackDevice = a.Value
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
	i := s.choices()
	if i == nil {
		return nil
	}
	source := i.Source
	if !i.MicActive() {
		source = "off"
	}
	rows := []ruleRow{{"source", "Source", source, false}, {"mode", "Processing", i.Mode, false}, {"monitor", "Monitor", i.Monitor, false}}
	rows = append(rows, ruleRow{"output", "Playback Device", i.PlaybackDevice, false})
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
			ruleRow{"record-vst", "Recording to VST", strconv.FormatBool(r.ToVST), true},
			ruleRow{"record-loop", "Loop Snippet", strconv.FormatBool(r.Loop), true},
			ruleRow{"record-start", "Start Recording", "Enter (live only)", false},
			ruleRow{"record-stop", "Stop Recording", "Enter (live only)", false},
			ruleRow{"snippet-play", "Play Snippet", "Enter (live only)", false},
			ruleRow{"snippet-stop", "Stop Playback", "Enter (live only)", false})
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
	if (row.key == "source" || row.key == "output") && (key == "enter" || key == " ") {
		s.openChoice(row.key)
		return
	}
	if key == "enter" && (row.key == "record-start" || row.key == "record-stop" || strings.HasPrefix(row.key, "snippet-")) {
		kind := startRecording
		if row.key == "record-stop" || row.key == "snippet-stop" {
			kind = stopRecording
		}
		if row.key == "snippet-play" {
			kind = playSnippet
		}
		s.queue(Action{Kind: kind, Revision: s.state.Revision})
		return
	}
	if row.key == "record-start" || row.key == "record-stop" || strings.HasPrefix(row.key, "snippet-") {
		return
	}
	if key != " " && key != "enter" {
		return
	}
	value := row.value
	switch row.key {
	case "record-tap":
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
	if a.Kind == editRule {
		s.queueEdit(a)
		return
	}
	if s.inflight != 0 || len(s.deferredEdits) != 0 {
		s.pending = "Settings queued; wait before this action"
		return
	}
	select {
	case s.actions <- a:
		s.pending = "Queued"
	default:
		s.pending = "A command is already queued"
	}
}
