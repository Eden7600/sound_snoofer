package control

import (
	"fmt"
	"strconv"
	"strings"

	"sound-snoofer/internal/config"
)

func EditIntent(i *config.Intent, a Action) error {
	if strings.HasPrefix(a.Row, "normal-") {
		a.Row = strings.TrimPrefix(a.Row, "normal-")
	}
	if strings.HasPrefix(a.Row, "vr-profile-") {
		if i.VRProfile == nil {
			return fmt.Errorf("VR profile unavailable")
		}
		switch strings.TrimPrefix(a.Row, "vr-profile-") {
		case "source":
			if a.Value == "off" {
				return fmt.Errorf("use Mic stack enablement to disable the stack")
			}
			i.VRProfile.Source = a.Value
		case "output":
			i.VRProfile.Playback = a.Value
		case "mode":
			i.VRProfile.Mode = a.Value
		case "monitor":
			i.VRProfile.Monitor = a.Value
		default:
			return fmt.Errorf("unknown VR setting")
		}
		return config.ValidateProfileChoices(*i.VRProfile)
	}
	switch a.Row {
	case "mic-mute", "speaker-mute", "vr-mic", "vr-playback", "defaults", "auto-recover":
		value, err := strconv.ParseBool(a.Value)
		if err != nil {
			return err
		}
		switch a.Row {
		case "mic-mute":
			i.MicMuted = value
		case "speaker-mute":
			i.PlaybackMuted = value
		case "vr-mic":
			i.PreferVRMic = value
		case "vr-playback":
			i.PreferVRPlayback = value
		case "defaults":
			i.ProtectDefaults = value
		case "auto-recover":
			i.AutoRecover = value
		}

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
		if a.Value == "off" {
			return fmt.Errorf("use Mic stack enablement to disable the stack")
		}
		i.Source = a.Value
	case "mode":
		i.Mode = a.Value
	case "monitor":
		i.Monitor = a.Value
	case "voice", "mic-stack":
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
	i.NormalizeRehearsal()
	return nil
}

const maxDeferredEdits = 64

type SettingEdit struct {
	Row, Value string
}

func EditsRow(action Action, row string) bool {
	if len(action.Edits) == 0 {
		return action.Row == row
	}
	for _, edit := range action.Edits {
		if edit.Row == row {
			return true
		}
	}
	return false
}

func EditBatch(intent *config.Intent, action Action) error {
	if len(action.Edits) == 0 {
		return EditIntent(intent, action)
	}
	if len(action.Edits) > maxDeferredEdits {
		return fmt.Errorf("too many queued settings")
	}
	for _, edit := range action.Edits {
		if err := EditIntent(intent, Action{Row: edit.Row, Value: edit.Value}); err != nil {
			return err
		}
	}
	return nil
}
