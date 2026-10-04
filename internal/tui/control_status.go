package tui

import (
	"fmt"
	"strconv"
	"strings"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/routing"
)

func intentValue(i *config.Intent, key string) string {
	if i == nil {
		return ""
	}
	switch key {
	case "source":
		if !i.MicActive() {
			return "off"
		}
		return i.Source
	case "mode":
		return i.Mode
	case "monitor":
		return i.Monitor
	case "output":
		return i.PlaybackDevice
	}
	if i.Recording != nil {
		switch key {
		case "record-tap":
			return i.Recording.MicTap
		case "record-mic":
			return strconv.FormatBool(i.Recording.MicEnabled)
		case "record-computer":
			return strconv.FormatBool(i.Recording.ComputerEnabled)
		case "record-vst":
			return strconv.FormatBool(i.Recording.ToVST)
		case "record-loop":
			return strconv.FormatBool(i.Recording.Loop)
		}
	}
	if strings.HasPrefix(key, "playback:") {
		return strconv.FormatBool(i.Playback[strings.TrimPrefix(key, "playback:")])
	}
	return ""
}
func controlValue(key, value string) string {
	if value == "true" {
		return "On"
	}
	if value == "false" {
		return "Off"
	}
	label := pretty(value)
	if key == "mode" && value == "element" {
		label = "Element"
	}
	if key == "monitor" && (value == "pre" || value == "post") {
		label += "-VST"
	}
	return label
}

// controlStatus returns an override (red) or unconfirmed application (yellow).
func (s screen) controlStatus(r ruleRow) (string, string) {
	if r.key == "record-start" || r.key == "record-stop" || strings.HasPrefix(r.key, "snippet-") {
		return "", ""
	}
	preferred := s.choices()
	effective := routing.ResolveIntent(preferred, s.state.Snapshot)
	value := intentValue(effective, r.key)
	if effective != nil {
		micUnavailable := !effective.MicActive()
		if s.state.Plan != nil && s.state.Plan.Topology != nil && s.state.Plan.Topology.Voice != nil {
			micUnavailable = micUnavailable || s.state.Plan.Topology.Voice.Effective == "unavailable"
		}
		if r.key == "record-mic" && effective.Recording != nil && (micUnavailable || effective.Recording.ToVST) {
			value = "false"
		}
		if r.key == "monitor" && micUnavailable {
			value = "off"
		}
	}

	if value != r.value {
		return "196", "! " + controlValue(r.key, r.value) + " → " + controlValue(r.key, value)
	}
	if s.inflight != 0 && editsRow(s.inflightAction, r.key) {
		return "226", "… pending"
	}
	for _, edit := range s.deferredEdits {
		if edit.Row == r.key {
			return "226", "… pending"
		}
	}
	if intentValue(s.state.Intent, r.key) != r.value {
		return "226", "… pending"
	}
	if s.state.Plan == nil || !s.state.Connected {
		return "226", "… pending"
	}
	t := s.state.Plan.Topology
	if t == nil {
		return "", ""
	}
	if r.key == "source" && t.Voice != nil && t.Voice.Effective != r.value {
		return "196", "! " + controlValue(r.key, r.value) + " → " + pretty(t.Voice.Effective)
	}
	if r.key == "output" && r.value != "" {
		actual := "unavailable"
		for _, op := range t.Operations {
			if op.Device != nil && op.Target == t.PlaybackTarget {
				actual = op.Device.Name
			}
		}
		if actual != r.value {
			return "196", "! fallback"
		}
	}
	for _, op := range t.Operations {
		if op.Change && controlDependsOn(r.key, op) {
			return "226", "… pending"
		}
	}
	return "", ""
}
func controlDependsOn(key string, op routing.Operation) bool {
	p := op.Parameter
	mic := strings.HasPrefix(p, "Strip[0].") || strings.HasPrefix(p, "Strip[1].") || strings.HasPrefix(p, "Strip[2].") || strings.HasPrefix(p, "Strip[6].")
	physical := strings.Contains(p, ".A")
	switch key {
	case "source":
		return (mic && (strings.HasSuffix(p, ".B2") || strings.HasSuffix(p, ".B3"))) || strings.HasPrefix(p, "Patch.") || strings.HasPrefix(op.Target, "input:")
	case "mode":
		return mic && (strings.HasSuffix(p, ".B2") || strings.HasSuffix(p, ".B3"))
	case "monitor":
		return (mic && physical) || strings.HasPrefix(p, "Recorder.A")
	case "output":
		return strings.HasPrefix(op.Target, "A") || physical
	case "record-tap", "record-mic":
		return mic && strings.HasSuffix(p, ".B1")
	case "record-computer":
		return !mic && strings.HasSuffix(p, ".B1")
	case "record-vst":
		return strings.HasPrefix(p, "Recorder.A") || strings.HasPrefix(p, "Recorder.B")
	case "record-loop":
		return p == "Recorder.mode.Loop"
	}
	if strings.HasPrefix(key, "playback:virtual:") {
		n, err := strconv.Atoi(strings.TrimPrefix(key, "playback:virtual:"))
		return err == nil && strings.HasPrefix(p, fmt.Sprintf("Strip[%d].A", 4+n))
	}
	return false
}
