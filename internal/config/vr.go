package config

import (
	"fmt"
	"regexp"
)

type VR struct {
	Input           int       `json:"input"`
	Headsets        []Headset `json:"headsets"`
	PlaybackDefault string    `json:"playback_default,omitempty"`
	CaptureDefault  string    `json:"capture_default,omitempty"`
}
type Headset struct {
	ID            string         `json:"id"`
	Label         string         `json:"label"`
	Microphone    string         `json:"microphone,omitempty"`
	Playback      string         `json:"playback,omitempty"`
	MicRegex      *regexp.Regexp `json:"-"`
	PlaybackRegex *regexp.Regexp `json:"-"`
}

func (v *VR) Validate() error {
	if v.Input == 0 {
		v.Input = 4
	}
	if v.Input < 4 || v.Input > 5 {
		return fmt.Errorf("VR input must be unreserved Potato input 4 or 5")
	}
	seen := map[string]bool{}
	for n := range v.Headsets {
		h := &v.Headsets[n]
		if h.ID == "" || seen[h.ID] || !regexp.MustCompile(`^[a-z0-9_-]+$`).MatchString(h.ID) {
			return fmt.Errorf("invalid or duplicate headset id")
		}
		seen[h.ID] = true
		if h.Microphone == "" && h.Playback == "" {
			return fmt.Errorf("headset needs a device matcher")
		}
		var err error
		if h.Microphone != "" {
			h.MicRegex, err = regexp.Compile(h.Microphone)
			if err != nil {
				return err
			}
		}
		if h.Playback != "" {
			h.PlaybackRegex, err = regexp.Compile(h.Playback)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
