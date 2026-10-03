package config

import (
	"fmt"
	"regexp"
)

// Studio owns A1 for the interface, inputs 1/2, the four input patch cells,
// and outputs whose names match Playback. Other occupied outputs are reserved.
type Studio struct {
	Recording           *Recording     `json:"recording,omitempty"`
	Voice               *Voice         `json:"voice,omitempty"`
	ASIOPattern         string         `json:"asio_pattern"`
	PresencePattern     string         `json:"presence_pattern"`
	Playback            []Candidate    `json:"playback"`
	FallbackMic         []Candidate    `json:"fallback_mic"`
	MovePlaybackRouting bool           `json:"move_playback_routing"`
	PlaybackSources     []string       `json:"playback_sources"`
	ASIORegex           *regexp.Regexp `json:"-"`
	PresenceRegex       *regexp.Regexp `json:"-"`
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
	if s.ASIOPattern == "" || s.PresencePattern == "" {
		return fmt.Errorf("studio requires asio_pattern and presence_pattern")
	}
	var e error
	s.ASIORegex, e = regexp.Compile(s.ASIOPattern)
	if e != nil {
		return fmt.Errorf("asio_pattern: %w", e)
	}
	s.PresenceRegex, e = regexp.Compile(s.PresencePattern)
	if e != nil {
		return fmt.Errorf("presence_pattern: %w", e)
	}
	if len(s.Playback) == 0 || len(s.FallbackMic) == 0 {
		return fmt.Errorf("studio requires playback and fallback_mic candidates")
	}
	for _, list := range [][]Candidate{s.Playback, s.FallbackMic} {
		for i := range list {
			if list[i].Driver != "wdm" || list[i].Pattern == "" {
				return fmt.Errorf("studio playback/fallback candidates require wdm and a nonempty pattern")
			}
			list[i].Regex, e = regexp.Compile(list[i].Pattern)
			if e != nil {
				return e
			}
		}
	}
	return nil
}
