package control

import (
	"strconv"

	"sound-snoofer/internal/routing"
)

// MuteStatus separates requested mute from known native readback.
type MuteStatus struct{ Requested, Muted, Known, Pending bool }

// ObserveMute evaluates every owned target, including the Element return.
func ObserveMute(s State, key string) (MuteStatus, bool) {
	parameters, requested, handled := routing.MuteTargets(s.Intent, s.Plan, key)
	result := MuteStatus{Requested: requested, Known: s.Connected && len(parameters) > 0}
	for _, p := range parameters {
		v, ok := s.Snapshot.Numbers[p]
		result.Known = result.Known && ok
		result.Muted = result.Muted || v == 1
		result.Pending = result.Pending || (requested && v != 1) || (key == "speaker-mute" && !requested && v != 0)
	}
	return result, handled
}

// ToggleMute changes requested state; native readback never becomes preference.
func ToggleMute(s State, key string) Action {
	_, requested, _ := routing.MuteTargets(s.Intent, s.Plan, key)
	return Action{Kind: Edit, Revision: s.Revision, Row: key, Value: strconv.FormatBool(!requested)}
}
