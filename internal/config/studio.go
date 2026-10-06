package config

import (
	"fmt"
	"regexp"
)

// Studio owns A1 for the interface, inputs 1/2, the four input patch cells,
// and outputs whose names match Playback. Other occupied outputs are reserved.
type Studio struct {
	ASIO                []ASIOInterface `json:"asio"`
	Recording           *Recording      `json:"recording,omitempty"`
	Voice               *Voice          `json:"voice,omitempty"`
	Playback            []Candidate     `json:"playback"`
	FallbackMic         []Candidate     `json:"fallback_mic"`
	MovePlaybackRouting bool            `json:"move_playback_routing"`
	PlaybackSources     []string        `json:"playback_sources"`
}

// ASIOInterface owns the clock and its available desk/lav input channels.
// A zero channel means that microphone is not attached to this interface.
type ASIOInterface struct {
	ASIOPattern     string         `json:"asio_pattern"`
	PresencePattern string         `json:"presence_pattern"`
	Inputs          [2]int         `json:"inputs"`
	ASIORegex       *regexp.Regexp `json:"-"`
	PresenceRegex   *regexp.Regexp `json:"-"`
}

func (s *Studio) OwnsASIO(name string) bool {
	for _, a := range s.ASIO {
		if name != "" && a.ASIORegex != nil && a.ASIORegex.MatchString(name) {
			return true
		}
	}
	return false
}

func (s *Studio) Validate() error {
	if s.Recording != nil {
		if s.Voice == nil {
			return fmt.Errorf("recording requires voice profile")
		}
		if e := s.Recording.Validate(); e != nil {
			return e
		}
	}
	if s.Voice != nil {
		if e := s.Voice.Validate(); e != nil {
			return e
		}
	}
	seen := map[string]bool{}
	for _, source := range s.PlaybackSources {
		if s.Voice != nil && source == "virtual:2" {
			return fmt.Errorf("AUX is reserved for Element")
		}
		if source != "virtual:1" && source != "virtual:2" && source != "virtual:3" {
			return fmt.Errorf("unsupported playback source %q; use virtual:1..3", source)
		}
		if seen[source] {
			return fmt.Errorf("duplicate playback source %q", source)
		}
		seen[source] = true
	}
	var e error
	for n := range s.ASIO {
		a := &s.ASIO[n]
		if a.ASIOPattern == "" || a.PresencePattern == "" {
			return fmt.Errorf("ASIO interface %d requires driver and presence patterns", n+1)
		}
		a.ASIORegex, e = regexp.Compile(a.ASIOPattern)
		if e != nil {
			return fmt.Errorf("ASIO interface %d: %w", n+1, e)
		}
		a.PresenceRegex, e = regexp.Compile(a.PresencePattern)
		if e != nil {
			return fmt.Errorf("ASIO presence %d: %w", n+1, e)
		}
		for _, channel := range a.Inputs {
			if channel < 0 || channel > 64 {
				return fmt.Errorf("ASIO input channel must be 0..64")
			}
		}
	}
	if len(s.Playback) == 0 || len(s.FallbackMic) == 0 {
		return fmt.Errorf("studio requires playback and fallback_mic candidates")
	}
	for group, list := range [][]Candidate{s.Playback, s.FallbackMic} {
		for i := range list {
			if (list[i].Driver != "wdm" && !(group == 0 && list[i].Driver == "asio")) || list[i].Pattern == "" {
				return fmt.Errorf("playback requires wdm/asio; fallback mic requires wdm; patterns must be nonempty")
			}
			list[i].Regex, e = regexp.Compile(list[i].Pattern)
			if e != nil {
				return e
			}
		}
	}
	return nil
}
