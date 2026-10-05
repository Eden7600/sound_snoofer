package routing

import (
	"fmt"
	"strings"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

type VoiceStatus struct {
	Strip            int    `json:"strip"`
	PreferredMode    string `json:"preferred_mode"`
	EffectiveMode    string `json:"effective_mode"`
	ProcessingReason string `json:"processing_reason,omitempty"`
	Preferred        string `json:"preferred"`
	Effective        string `json:"effective"`
	Reason           string `json:"reason,omitempty"`
	Monitor          string `json:"monitor"`
}

func addVoice(c config.Config, s model.Snapshot, p Plan) (Plan, error) {
	i := EffectiveIntent(c, s)
	if e := i.Validate(c); e != nil {
		return p, e
	}
	t := p.Topology
	t.MicStrips = ManagedMicStrips(c)
	if c.ProfileMicMissing {
		t.Unresolved = append(t.Unresolved, "No eligible microphone in active profile; no configured fallback")
	}
	v := &VoiceStatus{Preferred: i.Source, Effective: i.Source, Monitor: "off"}
	v.PreferredMode = c.VoiceIntent().Mode
	v.EffectiveMode = i.Mode
	if v.PreferredMode != v.EffectiveMode {
		v.ProcessingReason = s.ElementStatus() + "; using Direct"
	}
	t.Voice = v
	webcam, reasons := selectDevice(c.Studio.FallbackMic, "input", s.Devices)
	current := s.Assignments["input:3"]
	owned := current == ""
	for _, candidate := range c.Studio.FallbackMic {
		if candidate.Regex.MatchString(current) {
			owned = true
		}
	}
	if !owned {
		return p, fmt.Errorf("input:3 is occupied by unmanaged device %q", current)
	}
	if webcam != nil && i.MicActive() {
		t.Operations = append(t.Operations, Operation{Target: "input:3", Device: webcam, BeforeName: current, Change: current != webcam.Name})
	} else if !i.MicActive() {
		clear := &model.Device{Direction: "input", Driver: "wdm", Available: true}
		t.Operations = append(t.Operations, Operation{Target: "input:3", Device: clear, BeforeName: current, Change: current != ""})
	}
	source := 0
	if i.Source == "lav" {
		source = 1
	}
	if i.MicActive() && (i.Source == "webcam" || strings.HasPrefix(i.Source, "vr:") || !t.ASIOActive) {
		source = 2
		v.Effective = "webcam"
		if i.Source != "webcam" {
			v.Reason = "Volt unavailable; using webcam fallback"
		}
		if webcam == nil {
			source = -1
			v.Effective = "unavailable"
			v.Reason = strings.Join(reasons, "; ")
			t.Unresolved = append(t.Unresolved, "no eligible microphone: "+v.Reason)
		}
	}
	if !i.MicActive() {
		source = -1
		v.Effective = "off"
		v.Reason = "Mic stack disabled"
	}
	vrSource, vrID, err := addVRMic(c, s, t, source)
	if err != nil {
		return p, err
	}
	source = vrSource
	if vrID != "" {
		filtered := t.Unresolved[:0]
		for _, reason := range t.Unresolved {
			if !strings.HasPrefix(reason, "no eligible microphone:") {
				filtered = append(filtered, reason)
			}
		}
		t.Unresolved = filtered
		v.Effective = vrID
		v.Reason = "Headset microphone selected"
	}
	v.Strip = source
	desired := map[string]int{}
	for strip := 0; strip < 8; strip++ {
		for bus := 2; bus <= 3; bus++ {
			desired[fmt.Sprintf("Strip[%d].B%d", strip, bus)] = 0
		}
	}
	for _, strip := range ManagedMicStrips(c) {
		for bus := 1; bus <= 5; bus++ {
			desired[fmt.Sprintf("Strip[%d].A%d", strip, bus)] = 0
		}
	}
	rehearsal := i.Recording != nil && i.Recording.ToVST
	active := i.MicActive() && source >= 0 && !rehearsal
	if active {
		if i.Mode == "direct" {
			desired[fmt.Sprintf("Strip[%d].B3", source)] = 1
		} else {
			desired[fmt.Sprintf("Strip[%d].B2", source)] = 1
			desired["Strip[6].B3"] = 1
		}
	}
	monitor := -1
	switch {
	case rehearsal:
		if i.Monitor == "post" && t.PlaybackTarget != "" {
			monitor = 6
			v.Monitor = "Post-VST -> " + t.PlaybackTarget
		} else if i.Monitor == "pre" {
			v.Monitor = "Pre-VST tape -> " + t.PlaybackTarget
		}
	case !i.MicActive():
		v.Monitor = "inactive: voice disabled"
	case source < 0:
		v.Monitor = "inactive: mic unavailable"
	case i.Monitor == "off":
	case t.PlaybackTarget == "":
		v.Monitor = "inactive: playback unavailable"
	case i.Monitor == "pre":
		monitor = source
		v.Monitor = "pre -> " + t.PlaybackTarget
	case i.Mode == "element":
		monitor = 6
		v.Monitor = "post -> " + t.PlaybackTarget
	default:
		v.Monitor = "inactive: post requires Element mode"
	}
	if monitor >= 0 {
		desired[fmt.Sprintf("Strip[%d].%s", monitor, t.PlaybackTarget)] = 1
	}
	if c.SoundboardReserved {
		for bus := 1; bus <= 5; bus++ {
			desired[fmt.Sprintf("Strip[7].A%d", bus)] = 0
		}
		if c.SoundboardPolicy != nil {
			if routes := c.SoundboardPolicy(); routes != nil {
				if routes.Microphone {
					desired["Strip[7].B3"] = 1
				}
				if routes.Monitor && t.PlaybackTarget != "" {
					desired["Strip[7]."+t.PlaybackTarget] = 1
				}
			}
		}
	}
	// Stable strip/bus ordering places the processing feed before the AUX return.
	for strip := 0; strip < 8; strip++ {
		for _, letter := range []string{"B", "A"} {
			for bus := 1; bus <= 5; bus++ {
				param := fmt.Sprintf("Strip[%d].%s%d", strip, letter, bus)
				value, ok := desired[param]
				if !ok {
					continue
				}
				before, ok := s.Numbers[param]
				if !ok {
					return p, fmt.Errorf("snapshot missing %s", param)
				}
				t.Operations = append(t.Operations, Operation{Parameter: param, Value: value, BeforeValue: before, Change: before != float32(value)})
			}
		}
	}
	if c.Studio.Recording != nil {
		if e := addRecording(c, s, t, source); e != nil {
			return p, e
		}
	}
	// Master Off also disconnects microphone capture when B1 is otherwise
	// unmanaged or recorder-specific reconciliation is frozen.
	if !i.MicActive() {
		for _, strip := range ManagedMicStrips(c) {
			param := fmt.Sprintf("Strip[%d].B1", strip)
			before, ok := s.Numbers[param]
			if !ok {
				return p, fmt.Errorf("snapshot missing %s", param)
			}
			found := false
			for n := range t.Operations {
				if t.Operations[n].Parameter == param {
					t.Operations[n].Value = 0
					t.Operations[n].Change = before != 0
					found = true
				}
			}
			if !found {
				t.Operations = append(t.Operations, Operation{Parameter: param, BeforeValue: before, Value: 0, Change: before != 0})
			}
		}
	}
	buildVoiceTransition(c, s, t)
	return p, nil
}
