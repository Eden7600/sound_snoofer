package routing

import (
	"fmt"

	"sound-snoofer/internal/config"
)

// MuteTargets describes the existing requests, following the resolved playback destination.
func MuteTargets(i *config.Intent, p *Plan, key string) (parameters []string, requested, handled bool) {
	if i == nil {
		return nil, false, false
	}
	var t *Topology
	if p != nil {
		t = p.Topology
	}
	switch key {
	case "mic-mute":
		requested = i.MicMuted
		if t != nil && t.Voice != nil && t.Voice.Strip >= 0 && i.MicActive() {
			parameters = []string{fmt.Sprintf("Strip[%d].Mute", t.Voice.Strip), "Strip[6].Mute"}
		}
	case "speaker-mute":
		requested = i.PlaybackMuted
		if t != nil {
			bus := t.PlaybackTarget
			if len(bus) == 2 && bus[0] == 'A' && bus[1] >= '1' && bus[1] <= '5' {
				parameters = []string{fmt.Sprintf("Bus[%d].Mute", bus[1]-'1')}
			}
		}
	default:
		return nil, false, false
	}
	return parameters, requested, true
}
