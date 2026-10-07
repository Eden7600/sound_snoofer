package audio

import (
	"errors"
	"strconv"

	"sound-snoofer/internal/control"
	"sound-snoofer/internal/routing"
)

// MicMute is the Mic mute preference and how far Voicemeeter has applied it.
type MicMute struct {
	Known   bool // A preference is loaded.
	Muted   bool // The preference.
	Applied bool // Voicemeeter reads back the preference, or has nothing to mute.
}

// MicMute reports the current Mic mute preference.
func (i *Instance) MicMute() MicMute {
	i.mu.Lock()
	s := i.state
	i.mu.Unlock()
	if s.Intent == nil {
		return MicMute{}
	}
	// With the mic stack off there is nothing to mute in Voicemeeter.
	parameters, _, _ := routing.MuteTargets(s.Intent, s.Plan, "mic-mute")
	observed, _ := control.ObserveMute(s, "mic-mute")
	applied := len(parameters) == 0 || (observed.Known && !observed.Pending && observed.Muted == s.Intent.MicMuted)
	return MicMute{Known: true, Muted: s.Intent.MicMuted, Applied: applied}
}

// SetMicMute requests a Mic mute preference through the audio worker, as
// the Mic mute control does: it is saved, then applied to Voicemeeter.
// origin names the requesting surface for acknowledgements.
func (i *Instance) SetMicMute(muted bool, origin string) error {
	i.mu.Lock()
	revision := i.state.Revision
	i.mu.Unlock()
	a := control.Action{Origin: origin, Revision: revision, Kind: control.Edit, Row: "mic-mute", Value: strconv.FormatBool(muted)}
	select {
	case i.actions <- a:
		return nil
	default:
		return errors.New("audio queue full")
	}
}
