package routing

import (
	"fmt"
	"strings"
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

type RecordingStatus struct {
	Sources         int                     `json:"sources"`
	Mic             string                  `json:"mic"`
	ComputerSources []string                `json:"computer_sources"`
	Blocked         string                  `json:"blocked,omitempty"`
	Recorder        *model.RecorderSnapshot `json:"recorder"`
}

func recordCell(p string) bool { return strings.HasPrefix(p, "Strip[") && strings.HasSuffix(p, ".B1") }
func addRecording(c config.Config, s model.Snapshot, t *Topology, source int) error {
	i := c.VoiceIntent()
	r := i.Recording
	state := &RecordingStatus{Mic: "disabled", ComputerSources: append([]string{}, c.Studio.Recording.ComputerSources...), Recorder: s.Recorder}
	t.Recording = state
	state.Blocked = s.Recorder.Conflict()
	if state.Blocked != "" {
		t.Unresolved = append(t.Unresolved, state.Blocked)
		return nil
	}
	desired := [8]int{}
	if r.MicEnabled {
		switch {
		case !i.MicActive():
			state.Mic = "inactive: voice disabled"
		case source < 0:
			state.Mic = "inactive: microphone unavailable"
		case r.MicTap == "post" && i.Mode != "element":
			state.Mic = "inactive: Post requires Element voice mode"
		case r.MicTap == "post":
			desired[6] = 1
			state.Mic = "Post: AUX"
		default:
			desired[source] = 1
			state.Mic = "Pre: " + t.Voice.Effective
		}
	}
	if r.ComputerEnabled {
		for _, s := range c.Studio.Recording.ComputerSources {
			desired[5+int(s[len(s)-1]-'1')] = 1
		}
	}
	for strip, value := range desired {
		p := fmt.Sprintf("Strip[%d].B1", strip)
		before, ok := s.Numbers[p]
		if !ok {
			return fmt.Errorf("snapshot missing %s", p)
		}
		t.Operations = append(t.Operations, Operation{Parameter: p, Value: value, BeforeValue: before, Change: before != float32(value)})
		state.Sources += value
	}
	return nil
}
