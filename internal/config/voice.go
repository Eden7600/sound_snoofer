package config

import (
	"fmt"
	"strings"
)

type Voice struct {
	Enabled *bool  `json:"enabled,omitempty"`
	Source  string `json:"source,omitempty"`
	Mode    string `json:"mode,omitempty"`
	Monitor string `json:"monitor,omitempty"`
}
type Intent struct {
	VRProfile        *ProfileChoices `json:"vr_profile,omitempty"`
	MicMuted         bool            `json:"mic_muted,omitempty"`
	PlaybackMuted    bool            `json:"playback_muted,omitempty"`
	BusMuted         [2]bool         `json:"bus_muted,omitempty"`
	PreferVRMic      bool            `json:"prefer_vr_mic,omitempty"`
	PreferVRPlayback bool            `json:"prefer_vr_playback,omitempty"`
	ProtectDefaults  bool            `json:"protect_defaults,omitempty"`
	AutoRecover      bool            `json:"auto_recover,omitempty"`

	PlaybackDevice string            `json:"playback_device,omitempty"`
	Recording      *RecordingChoices `json:"recording,omitempty"`
	Version        int               `json:"version"`
	Enabled        bool              `json:"enabled"`
	Source         string            `json:"source"`
	Mode           string            `json:"mode"`
	Monitor        string            `json:"monitor"`
	Playback       map[string]bool   `json:"playback"`
}

func (v *Voice) Validate() error {
	if v.Source == "" {
		v.Source = "desk"
	}
	if v.Mode == "" {
		v.Mode = "element"
	}
	if v.Monitor == "" {
		v.Monitor = "off"
	}
	return validateChoices(v.Source, v.Mode, v.Monitor)
}
func validateChoices(source, mode, monitor string) error {
	if source != "auto" && source != "off" && source != "desk" && source != "lav" && source != "webcam" && !strings.HasPrefix(source, "vr:") {
		return fmt.Errorf("invalid voice source %q", source)
	}
	if mode != "direct" && mode != "element" {
		return fmt.Errorf("invalid voice mode %q", mode)
	}
	if monitor != "off" && monitor != "pre" && monitor != "post" {
		return fmt.Errorf("invalid monitor mode %q", monitor)
	}
	return nil
}
func (c Config) VoiceIntent() *Intent {
	if c.Studio == nil || c.Studio.Voice == nil {
		return nil
	}
	if c.Intent != nil {
		i := c.Intent.Clone()
		i.normalizeSource()
		i.NormalizeRecording(c)
		return i
	}
	v := c.Studio.Voice
	i := &Intent{Version: 1, Enabled: v.Enabled == nil || *v.Enabled, Source: v.Source, Mode: v.Mode, Monitor: v.Monitor, Playback: map[string]bool{}}
	for _, s := range c.Studio.PlaybackSources {
		i.Playback[s] = true
	}
	i.NormalizeRecording(c)
	i.normalizeSource()
	return i
}

// normalizeSource interprets legacy Off without discarding a disabled target.
func (i *Intent) normalizeSource() {
	if i.Source == "off" {
		i.Enabled = false
		i.Source = "auto"
	}
	if i.VRProfile != nil && i.VRProfile.Source == "off" {
		i.Enabled = false
		i.VRProfile.Source = "auto"
	}
}
func (i *Intent) MicActive() bool { return i != nil && i.Enabled && i.Source != "off" }
func (i *Intent) Clone() *Intent {
	if i == nil {
		return nil
	}
	n := *i
	if i.VRProfile != nil {
		p := *i.VRProfile
		n.VRProfile = &p
	}
	if i.Recording != nil {
		r := *i.Recording
		n.Recording = &r
	}
	n.Playback = map[string]bool{}
	for k, v := range i.Playback {
		n.Playback[k] = v
	}
	return &n
}
func (i Intent) Validate(c Config) error {
	if i.VRProfile != nil {
		if err := ValidateProfileChoices(*i.VRProfile); err != nil {
			return err
		}
	}
	if strings.HasPrefix(i.Source, "vr:") {
		found := false
		if c.VR != nil {
			for _, h := range c.VR.Headsets {
				found = found || i.Source == "vr:"+h.ID
			}
		}
		if !found {
			return fmt.Errorf("unknown VR microphone")
		}
	}

	if i.Recording != nil {
		if c.Studio == nil || c.Studio.Recording == nil {
			return fmt.Errorf("saved recording choices require recording profile")
		}
		if i.Recording.MicTap != "pre" && i.Recording.MicTap != "post" {
			return fmt.Errorf("recording mic_tap must be pre or post")
		}
	}
	if c.Studio == nil || c.Studio.Voice == nil {
		return fmt.Errorf("saved choices require voice profile")
	}
	if i.PlaybackDevice != "" {
		matched := c.Studio.ASIOPlayback && c.Studio.ASIORegex != nil && c.Studio.ASIORegex.MatchString(i.PlaybackDevice)
		for _, candidate := range c.PlaybackCandidates() {
			if candidate.Regex != nil && candidate.Regex.MatchString(i.PlaybackDevice) {
				matched = true
			}
		}
		if !matched {
			return fmt.Errorf("playback device does not match configured playback rules")
		}
	}
	if i.Version != 1 {
		return fmt.Errorf("saved choices version must be 1")
	}
	if e := validateChoices(i.Source, i.Mode, i.Monitor); e != nil {
		return e
	}
	if len(i.Playback) != len(c.Studio.PlaybackSources) {
		return fmt.Errorf("saved playback rules differ from configuration; reset saved choices")
	}
	for _, s := range c.Studio.PlaybackSources {
		if _, ok := i.Playback[s]; !ok {
			return fmt.Errorf("saved choices missing playback rule %s", s)
		}
	}
	return nil
}
