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
	FallbackMic         []Candidate     `json:"fallback_mic,omitempty"`
	MovePlaybackRouting bool            `json:"move_playback_routing"`
	PlaybackSources     []string        `json:"playback_sources"`
	// Microphones in input order; see Microphone. Absent in configurations
	// from before microphones were configurable, which Validate fills.
	Microphones       []Microphone `json:"microphones,omitempty"`
	legacyMicrophones bool
	Outputs           []Output `json:"outputs,omitempty"`
}

// ASIOInterface owns the clock and the input channels of the interface
// microphones it carries; see MicInputs.
type ASIOInterface struct {
	ASIOPattern     string `json:"asio_pattern,omitempty"`
	PresencePattern string `json:"presence_pattern,omitempty"`
	// Identity alternatives to the patterns: the ASIO driver CLSID and the
	// presence input's Windows endpoint ID, with names as labels.
	ASIOID        string         `json:"asio_id,omitempty"`
	ASIOName      string         `json:"asio_name,omitempty"`
	PresenceID    string         `json:"presence_id,omitempty"`
	PresenceName  string         `json:"presence_name,omitempty"`
	Inputs        MicInputs      `json:"inputs,omitempty"`
	ASIORegex     *regexp.Regexp `json:"-"`
	PresenceRegex *regexp.Regexp `json:"-"`
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
	if e := s.validateOutputs(); e != nil {
		return e
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
		a.ASIORegex, e = compileMatcher(a.ASIOPattern, a.ASIOID, a.ASIOName, fmt.Sprintf("ASIO interface %d driver", n+1))
		if e != nil {
			return e
		}
		a.PresenceRegex, e = compileMatcher(a.PresencePattern, a.PresenceID, a.PresenceName, fmt.Sprintf("ASIO interface %d presence", n+1))
		if e != nil {
			return e
		}
	}
	if len(s.Playback) == 0 {
		return fmt.Errorf("studio requires playback candidates")
	}
	for i := range s.Playback {
		if s.Playback[i].Driver != "wdm" && s.Playback[i].Driver != "asio" {
			return fmt.Errorf("playback requires wdm/asio")
		}
		if e = s.Playback[i].compile(fmt.Sprintf("playback %d", i+1)); e != nil {
			return e
		}
	}
	if e = s.validateMicrophones(); e != nil {
		return e
	}
	if s.Voice != nil {
		if s.Voice.Source == "" {
			s.Voice.Source = "auto"
			if _, _, ok := s.Microphone("desk"); ok {
				s.Voice.Source = "desk"
			}
		}
		if !(Config{Studio: s}).validMicrophoneChoice(s.Voice.Source) {
			return fmt.Errorf("voice source %q is not a configured microphone", s.Voice.Source)
		}
	}
	return nil
}
