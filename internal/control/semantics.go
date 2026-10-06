package control

import (
	"fmt"
	"strconv"
	"strings"

	"sound-snoofer/internal/routing"
)

// MicStrips includes the configured VR strip even after Source Off.
func MicStrips(s State) []int {
	if s.Plan != nil && s.Plan.Topology != nil && len(s.Plan.Topology.MicStrips) > 0 {
		return s.Plan.Topology.MicStrips
	}
	strips := []int{0, 1, 2, 6}
	if s.Plan != nil && s.Plan.Topology != nil && s.Plan.Topology.Voice != nil && s.Plan.Topology.Voice.Strip >= 0 {
		strips = append(strips, s.Plan.Topology.Voice.Strip)
	}
	return strips
}

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

// DependsOn identifies which control is affected by a planned write.
func DependsOn(s State, key string, op routing.Operation) bool {
	p := op.Parameter
	mic := false
	strips := MicStrips(s)
	for _, strip := range strips {
		if strip >= 0 && strings.HasPrefix(p, fmt.Sprintf("Strip[%d].", strip)) {
			mic = true
		}
	}
	switch key {
	case "source":
		return (mic && (strings.HasSuffix(p, ".B2") || strings.HasSuffix(p, ".B3"))) || strings.HasPrefix(p, "Patch.") || strings.HasPrefix(op.Target, "input:")
	case "output":
		return strings.HasPrefix(op.Target, "A") || strings.Contains(p, ".A")
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
	if strings.HasPrefix(key, "playback:virtual:") {
		n, e := strconv.Atoi(strings.TrimPrefix(key, "playback:virtual:"))
		return e == nil && strings.HasPrefix(p, fmt.Sprintf("Strip[%d].A", 4+n))
	}
	return false
}
