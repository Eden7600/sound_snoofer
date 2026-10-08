package config

import (
	"fmt"
	"strings"
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
	Microphones []string  `json:"microphones"`
	Activity    *Activity `json:"activity,omitempty"`
}

// DefaultProfiles is the Normal source priority for configurations written
// before profiles existed. The embedded default configuration states the same
// priority explicitly.
func DefaultProfiles() *Profiles {
	return &Profiles{Microphones: []string{"lav", "webcam"}}
}

// ProfilePolicy is supplied by an enabled dependent plugin, never discovered by audio.
type ProfilePolicy struct {
	Devices     VR             `json:"devices"`
	Microphones []string       `json:"microphones"`
	Playback    []Candidate    `json:"playback"`
	Choices     ProfileChoices `json:"choices"`
	// Process is the executable whose presence activates the profile. Empty
	// means DefaultVRProcess.
	Process string `json:"process,omitempty"`
	Running bool   `json:"-"`
	Known   bool   `json:"-"`
}

func (p *ProfilePolicy) Validate() error {
	if p.Process != "" {
		if err := ValidateProcessName(p.Process); err != nil {
			return fmt.Errorf("process: %w", err)
		}
	}
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
		// Studio microphones are checked against the audio configuration when
		// planning; the VR plugin only checks their form.
		if id == "off" || (microphoneID.MatchString(id) && !strings.HasPrefix(id, "vr")) {
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
		if c.Driver != "wdm" && c.Driver != "asio" {
			return fmt.Errorf("playback requires wdm/asio")
		}
		if err := c.compile("VR playback"); err != nil {
			return err
		}
	}
	return nil
}
func ValidateProfileChoices(p ProfileChoices) error {
	if p.Source == "" {
		return fmt.Errorf("profile source must be auto, off or a configured source")
	}
	return validateChoices(p.Source, p.Mode, p.Monitor)
}

// Activity opts automatic microphone selection into skipping microphones
// that deliver no signal. Absent, selection ignores signal entirely.
type Activity struct {
	// Check is how many leading available Normal priority options stay wired
	// and metered.
	Check int `json:"check"`
	// SilenceDB is the pre-fader peak below which a microphone counts as
	// silent; zero means -70 dBFS.
	SilenceDB float64 `json:"silence_db,omitempty"`
	// SilentAfterS is how long a microphone must stay silent before Auto
	// skips it; zero means 10 seconds.
	SilentAfterS int `json:"silent_after_s,omitempty"`
}

// Validate fills defaults and checks ranges.
func (a *Activity) Validate() error {
	if a.SilenceDB == 0 {
		a.SilenceDB = -70
	}
	if a.SilentAfterS == 0 {
		a.SilentAfterS = 10
	}
	if a.Check < 1 || a.Check > 4 {
		return fmt.Errorf("activity check must be 1-4")
	}
	if a.SilenceDB < -120 || a.SilenceDB > -20 {
		return fmt.Errorf("activity silence_db must be between -120 and -20")
	}
	if a.SilentAfterS < 2 || a.SilentAfterS > 600 {
		return fmt.Errorf("activity silent_after_s must be 2-600")
	}
	return nil
}
