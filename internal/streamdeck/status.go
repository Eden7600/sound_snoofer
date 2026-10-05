package streamdeck

import (
	"fmt"
	"strings"

	"sound-snoofer/internal/control"
	"sound-snoofer/internal/routing"
)

func keyValue(s control.State, key string) string {
	if key == "open-controls" {
		return "OPEN"
	}
	if !s.Live {
		return "PREVIEW"
	}
	if strings.HasPrefix(key, "media-") {
		return "PRESS"
	}
	if feedback, ok := s.Feedback[key]; ok {
		if feedback.Kind == control.NoticeError {
			return "ERROR"
		}
		if feedback.Kind == control.NoticePending {
			return "PENDING"
		}
	}
	if !s.Connected {
		return "UNAVAIL"
	}
	switch key {
	case "record-start", "record-stop", "snippet-play", "snippet-stop":
		if s.Recorder.State() == "Unknown" {
			return "UNAVAIL"
		}
		return s.Recorder.State()
	case "engine-restart":
		return "PRESS"
	}
	if s.Intent == nil {
		return "UNAVAIL"
	}
	if strings.HasPrefix(key, "record-") && s.Intent.Recording == nil {
		return "UNAVAIL"
	}
	if value, handled := muteValue(s, key); handled {
		return value
	}
	if s.Plan != nil && s.Plan.Topology != nil {
		for _, op := range s.Plan.Topology.Operations {
			if op.Change && dependsOn(s, key, op) {
				return "PENDING"
			}
		}
	}
	return Value(s, key)
}

func muteValue(s control.State, key string) (string, bool) {
	parameters := []string{}
	requested := false
	switch key {
	case "a1-mute", "a2-mute":
		n := int(key[1] - '1')
		parameters = append(parameters, fmt.Sprintf("Bus[%d].Mute", n))
		requested = s.Intent.BusMuted[n]
	case "speaker-mute":
		requested = s.Intent.PlaybackMuted
		if s.Plan == nil || s.Plan.Topology == nil || len(s.Plan.Topology.PlaybackTarget) != 2 {
			return "UNAVAIL", true
		}
		n := int(s.Plan.Topology.PlaybackTarget[1] - '1')
		parameters = append(parameters, fmt.Sprintf("Bus[%d].Mute", n))
		if n >= 0 && n < len(s.Intent.BusMuted) {
			requested = requested || s.Intent.BusMuted[n]
		}
	case "mic-mute":
		requested = s.Intent.MicMuted
		if s.Plan == nil || s.Plan.Topology == nil || s.Plan.Topology.Voice == nil || s.Plan.Topology.Voice.Strip < 0 {
			return "UNAVAIL", true
		}
		parameters = append(parameters, fmt.Sprintf("Strip[%d].Mute", s.Plan.Topology.Voice.Strip), "Strip[6].Mute")
	default:
		return "", false
	}
	anyMuted := false
	for _, p := range parameters {
		v, ok := s.Snapshot.Numbers[p]
		if !ok {
			return "UNAVAIL", true
		}
		if requested && v != 1 {
			return "PENDING", true
		}
		anyMuted = anyMuted || v == 1
	}
	// Preserve native manual mute visibility even when Snoofer did not request it.
	if anyMuted {
		return "On", true
	}
	return "Off", true
}

func dependsOn(s control.State, key string, op routing.Operation) bool {
	p := op.Parameter
	mic := false
	strips := []int{0, 1, 2, 6}
	if s.Plan != nil && s.Plan.Topology != nil && s.Plan.Topology.Voice != nil {
		strips = append(strips, s.Plan.Topology.Voice.Strip)
	}
	for _, strip := range strips {
		if strip >= 0 && strings.HasPrefix(p, fmt.Sprintf("Strip[%d].", strip)) {
			mic = true
		}
	}
	switch key {
	case "monitor":
		return (mic && strings.Contains(p, ".A")) || strings.HasPrefix(p, "Recorder.A")
	case "mode":
		return mic && (strings.HasSuffix(p, ".B2") || strings.HasSuffix(p, ".B3"))
	case "record-mic", "record-tap":
		return mic && strings.HasSuffix(p, ".B1")
	case "record-computer":
		return (strings.HasPrefix(p, "Strip[5].") || strings.HasPrefix(p, "Strip[7].")) && strings.HasSuffix(p, ".B1")
	case "record-loop":
		return p == "Recorder.mode.Loop"
	case "record-vst":
		return strings.HasPrefix(p, "Recorder.A") || strings.HasPrefix(p, "Recorder.B")
	}
	return false
}
