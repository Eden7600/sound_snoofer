package config

import (
	"fmt"
	"regexp"
)

// ProfileChoices contains independent persisted source, playback and processing choices.
type ProfileChoices struct {
	Source   string `json:"source"`
	Playback string `json:"playback,omitempty"`
	Mode     string `json:"mode"`
	Monitor  string `json:"monitor"`
}

// Profiles configures the Normal source priority. Playback retains Studio.Playback.
type Profiles struct {
	Microphones []string `json:"microphones"`
}

// ProfilePolicy is supplied by an enabled dependent plugin, never discovered by audio.
type ProfilePolicy struct {
	Devices     VR             `json:"devices"`
	Microphones []string       `json:"microphones"`
	Playback    []Candidate    `json:"playback"`
	Choices     ProfileChoices `json:"choices"`
	Running     bool           `json:"-"`
	Known       bool           `json:"-"`
}

func (p *ProfilePolicy) Validate() error {
	if err := p.Devices.Validate(); err != nil {
		return err
	}
	if err := ValidateProfileChoices(p.Choices); err != nil {
		return err
	}
	if len(p.Microphones) == 0 || len(p.Playback) == 0 {
		return fmt.Errorf("profile requires microphone and playback priorities")
	}
	for n, id := range p.Microphones {
		if id == "normal" {
			if n != len(p.Microphones)-1 {
				return fmt.Errorf("Normal fallback must be last")
			}
			continue
		}
		if id == "desk" || id == "lav" || id == "webcam" || id == "off" {
			continue
		}
		found := false
		for _, h := range p.Devices.Headsets {
			found = found || id == "vr:"+h.ID
		}
		if !found {
			return fmt.Errorf("unknown microphone priority %s", id)
		}
	}
	for n := range p.Playback {
		c := &p.Playback[n]
		if c.Driver == "normal" {
			if n != len(p.Playback)-1 || c.Pattern != "" {
				return fmt.Errorf("Normal fallback must be last and have no pattern")
			}
			continue
		}
		if c.Driver != "wdm" || c.Pattern == "" {
			return fmt.Errorf("playback requires wdm and pattern")
		}
		re, err := regexp.Compile(c.Pattern)
		if err != nil {
			return err
		}
		c.Regex = re
	}
	return nil
}
func ValidateProfileChoices(p ProfileChoices) error {
	if p.Source == "" {
		return fmt.Errorf("profile source must be auto, off or a configured source")
	}
	return validateChoices(p.Source, p.Mode, p.Monitor)
}
