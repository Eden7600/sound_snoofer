package routing

import (
	"fmt"

	"sound-snoofer/internal/config"
)

// MuteTargets describes the existing requests, including fixed-bus composition.
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
	case "speaker-mute", "a1-mute", "a2-mute":
		bus := ""
		if key == "speaker-mute" {
			if t != nil {
				bus = t.PlaybackTarget
			}
		} else {
			bus = fmt.Sprintf("A%c", key[1])
		}
		if len(bus) == 2 && bus[0] == 'A' && bus[1] >= '1' && bus[1] <= '5' {
			n := int(bus[1] - '1')
			parameters = []string{fmt.Sprintf("Bus[%d].Mute", n)}
			if n < len(i.BusMuted) {
				requested = i.BusMuted[n]
			}
			requested = requested || (t != nil && t.PlaybackTarget == bus && i.PlaybackMuted)
		}
	default:
		return nil, false, false
	}
	return parameters, requested, true
}
