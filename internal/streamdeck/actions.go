package streamdeck

import (
	"sound-snoofer/internal/control"
	"sound-snoofer/internal/controller"
	"strconv"
)

func Value(s control.State, key string) string {
	i := s.Intent
	if i == nil {
		return "?"
	}
	v := false
	switch key {
	case "vr-mic":
		v = i.PreferVRMic
	case "vr-playback":
		v = i.PreferVRPlayback
	case "defaults":
		v = i.ProtectDefaults
	case "mic-mute":
		v = i.MicMuted
	case "speaker-mute":
		v = i.PlaybackMuted
	case "a1-mute":
		v = i.BusMuted[0]
	case "a2-mute":
		v = i.BusMuted[1]
	case "monitor":
		return i.Monitor
	case "mode":
		return i.Mode
	default:
		if i.Recording == nil {
			return "?"
		}
		switch key {
		case "record-mic":
			v = i.Recording.MicEnabled
		case "record-computer":
			v = i.Recording.ComputerEnabled
		case "record-loop":
			v = i.Recording.Loop
		case "record-vst":
			v = i.Recording.ToVST
		case "record-tap":
			return i.Recording.MicTap
		}
	}
	if v {
		return "On"
	}
	return "Off"
}
func Action(e Event, s control.State) (control.Action, string) {
	a := control.Action{Origin: "streamdeck", Revision: s.Revision}
	key := ""
	if e.Encoder >= 0 {
		if e.Encoder > 2 {
			return a, ""
		}
		target := []string{"A1", "A2", "mic"}[e.Encoder]
		if e.Delta != 0 {
			a.Kind = control.Gain
			a.Target = target
			a.Identity = controller.GainIdentity(s.Plan, s.Snapshot, target)
			a.Delta = float32(e.Delta)
			return a, "audio"
		}
		key = []string{"a1-mute", "a2-mute", "mic-mute"}[e.Encoder]
	} else {
		if e.Binding != "" {
			key = e.Binding
		} else if e.Key < 0 || e.Key >= len(bindings) {
			return a, ""
		}
		if key == "" {
			key = bindings[e.Key]
		}
	}
	switch key {
	case "":
		return a, ""
	case "record-toggle":
		if !s.Connected {
			return a, ""
		}
		switch s.Recorder.State() {
		case "Stopped":
			a.Kind = control.RecordStart
		case "Recording":
			a.Kind = control.RecordStop
		case "Paused":
			if s.Recorder.Values["Recorder.record"] != 1 {
				return a, ""
			}
			a.Kind = control.RecordStop
		default:
			return a, ""
		}
	case "speaker-mute", "a1-mute", "a2-mute":
		if s.Intent == nil {
			return a, ""
		}
		target := ""
		if s.Plan != nil && s.Plan.Topology != nil {
			target = s.Plan.Topology.PlaybackTarget
		}
		bus := ""
		if key == "a1-mute" {
			bus = "A1"
		}
		if key == "a2-mute" {
			bus = "A2"
		}
		a.Kind = control.Edit
		if key == "speaker-mute" || (bus != "" && bus == target) {
			if target == "" {
				return a, ""
			}
			muted := s.Intent.PlaybackMuted
			busRow := ""
			if target == "A1" {
				muted = muted || s.Intent.BusMuted[0]
				busRow = "a1-mute"
			}
			if target == "A2" {
				muted = muted || s.Intent.BusMuted[1]
				busRow = "a2-mute"
			}
			a.Edits = []control.SettingEdit{{Row: "speaker-mute", Value: strconv.FormatBool(!muted)}}
			if busRow != "" {
				a.Edits = append(a.Edits, control.SettingEdit{Row: busRow, Value: "false"})
			}
		} else {
			a.Row = key
			a.Value = strconv.FormatBool(Value(s, key) != "On")
		}
	case "engine-restart":
		a.Kind = control.Restart
	case "record-start":
		a.Kind = control.RecordStart
	case "record-stop", "snippet-stop":
		a.Kind = control.RecordStop
	case "snippet-play":
		a.Kind = control.SnippetPlay
	case "open-controls", "media-prev", "media-play", "media-next", "media-stop":
		return a, key
	default:
		a.Kind = control.Edit
		a.Row = key
		value := Value(s, key)
		a.Value = strconv.FormatBool(value != "On")
		switch key {
		case "monitor":
			switch value {
			case "off":
				a.Value = "pre"
			case "pre":
				a.Value = "post"
			default:
				a.Value = "off"
			}
		case "mode":
			a.Value = "element"
			if value == "element" {
				a.Value = "direct"
			}
		case "record-tap":
			a.Value = "post"
			if value == "post" {
				a.Value = "pre"
			}
		}
	}
	return a, "audio"
}
