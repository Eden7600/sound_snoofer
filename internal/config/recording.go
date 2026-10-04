package config

import "fmt"

type Recording struct {
	ComputerSources []string `json:"computer_sources"`
}
type RecordingChoices struct {
	TapeRoutingManaged bool   `json:"tape_routing_managed,omitempty"`
	ToVST              bool   `json:"to_vst,omitempty"`
	Loop               bool   `json:"loop,omitempty"`
	MicEnabled         bool   `json:"mic_enabled"`
	MicTap             string `json:"mic_tap"`
	ComputerEnabled    bool   `json:"computer_enabled"`
}

func (r *Recording) Validate() error {
	if r.ComputerSources == nil {
		r.ComputerSources = []string{"virtual:1"}
	}
	seen := map[string]bool{}
	for _, s := range r.ComputerSources {
		if (s != "virtual:1" && s != "virtual:3") || seen[s] {
			return fmt.Errorf("invalid or duplicate recording computer source %q; AUX is reserved", s)
		}
		seen[s] = true
	}
	return nil
}
func (i *Intent) NormalizeRecording(c Config) {
	if i.Recording == nil && c.Studio != nil && c.Studio.Recording != nil {
		i.Recording = &RecordingChoices{MicTap: "pre"}
	}
	i.NormalizeRecordingStage()
}

// NormalizeRecordingStage maintains rehearsal ownership; Post remains a preference.
func (i *Intent) NormalizeRecordingStage() {
	if i.Recording != nil && i.Recording.ToVST {
		i.Recording.TapeRoutingManaged = true
	}
	if i.Recording != nil && (!i.MicActive() || i.Mode != "element") {
		i.Recording.ToVST = false
	}
}
