package model

import "fmt"

type RecorderSnapshot struct {
	Values map[string]float32 `json:"values"`
	Error  string             `json:"error,omitempty"`
}

func RecorderParameters() []string {
	p := []string{"Recorder.record", "Recorder.stop", "Recorder.pause", "Recorder.play", "Recorder.mode.recbus", "Recorder.Channel", "Recorder.mode.MultiTrack", "Recorder.B1", "Recorder.B2", "Recorder.B3", "Bus[5].Mute"}
	for i := 0; i < 8; i++ {
		p = append(p, fmt.Sprintf("Recorder.ArmBus[%d]", i))
	}
	return p
}
func (r *RecorderSnapshot) State() string {
	if r == nil || r.Error != "" {
		return "Unknown"
	}
	for _, p := range RecorderParameters() {
		if _, ok := r.Values[p]; !ok {
			return "Unknown"
		}
	}
	if r.Values["Recorder.pause"] == 1 {
		return "Paused"
	}
	if r.Values["Recorder.record"] == 1 {
		return "Recording"
	}
	if r.Values["Recorder.play"] == 1 {
		return "Playing"
	}
	if r.Values["Recorder.stop"] == 1 {
		return "Stopped"
	}
	return "Unknown"
}

type RecorderSetting struct {
	Parameter string
	Value     int
}

func RecorderSetup() []RecorderSetting {
	p := []RecorderSetting{{"Recorder.mode.recbus", 1}, {"Recorder.Channel", 2}, {"Recorder.mode.MultiTrack", 0}, {"Recorder.B1", 0}, {"Recorder.B2", 0}, {"Recorder.B3", 0}}
	for i := 0; i < 8; i++ {
		v := 0
		if i == 5 {
			v = 1
		}
		p = append(p, RecorderSetting{fmt.Sprintf("Recorder.ArmBus[%d]", i), v})
	}
	return p
}
func (r *RecorderSnapshot) Ready() bool {
	if r.State() == "Unknown" {
		return false
	}
	for _, p := range RecorderSetup() {
		if r.Values[p.Parameter] != float32(p.Value) {
			return false
		}
	}
	return true
}
func (r *RecorderSnapshot) Conflict() string {
	if r.State() == "Unknown" {
		return "recorder state unavailable"
	}
	if r.State() != "Stopped" && !r.Ready() {
		return "active recorder configuration conflicts with B1 capture; stop first"
	}
	return ""
}
