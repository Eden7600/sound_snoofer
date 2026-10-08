package config

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Microphone is a user-defined microphone. Without Devices it is an
// interface microphone, fed by the channels each interface maps to its ID;
// with Devices it is a Windows input device chosen by priority. Its position
// in Studio.Microphones is its hardware input: the first uses input 1.
type Microphone struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Devices []Candidate `json:"devices,omitempty"`
	// Ready asserts the microphone is usable whenever it is present: it is
	// never metered, so silence (a hardware mute) never makes Auto skip it.
	Ready bool `json:"ready,omitempty"`
}

// IsDevice reports whether the microphone is a Windows input device.
func (m Microphone) IsDevice() bool { return len(m.Devices) > 0 }

// MaxMicrophones is the number of Potato hardware inputs. The VR input, when
// configured, must lie beyond the microphones; planning enforces that.
const MaxMicrophones = 5

var microphoneID = regexp.MustCompile(`^[a-z0-9-]{1,24}$`)

// MicInputs maps interface microphone IDs to one channel (mono, fed to both
// sides of the strip) or two channels (stereo left and right), numbered
// from 1. The legacy form is an array [desk, lav] where zero means absent.
type MicInputs map[string][]int

func (m *MicInputs) UnmarshalJSON(b []byte) error {
	var legacy [2]int
	if err := json.Unmarshal(b, &legacy); err == nil {
		*m = MicInputs{}
		for n, id := range []string{"desk", "lav"} {
			if legacy[n] != 0 {
				(*m)[id] = []int{legacy[n]}
			}
		}
		return nil
	}
	var mapped map[string][]int
	if err := json.Unmarshal(b, &mapped); err != nil {
		return fmt.Errorf("inputs must map microphone IDs to channels: %w", err)
	}
	*m = mapped
	return nil
}

// Left and Right are a microphone's channels on the strip; zero when the
// interface does not map it.
func (m MicInputs) Left(id string) int {
	if channels := m[id]; len(channels) > 0 {
		return channels[0]
	}
	return 0
}

func (m MicInputs) Right(id string) int {
	channels := m[id]
	if len(channels) == 2 {
		return channels[1]
	}
	return m.Left(id)
}

// legacyMicrophones is the microphone set of configurations written before
// microphones were configurable: desk and lav on the interface, and the
// webcam from fallback_mic.
func legacyMicrophones(fallback []Candidate) []Microphone {
	mics := []Microphone{{ID: "desk", Name: "Desk"}, {ID: "lav", Name: "Lavalier"}}
	if len(fallback) > 0 {
		mics = append(mics, Microphone{ID: "webcam", Name: "Webcam", Devices: fallback})
	}
	return mics
}

// Mics lists the microphones in input order. A studio without configured
// microphones has the legacy set, even before Validate fills it.
func (s *Studio) Mics() []Microphone {
	if s.Microphones == nil {
		return legacyMicrophones(s.FallbackMic)
	}
	return s.Microphones
}

// Microphone returns the microphone with id and its position (input index).
func (s *Studio) Microphone(id string) (Microphone, int, bool) {
	for n, m := range s.Mics() {
		if m.ID == id {
			return m, n, true
		}
	}
	return Microphone{}, -1, false
}

// MicrophoneIDs lists the configured microphone IDs in input order.
func (s *Studio) MicrophoneIDs() []string {
	ids := []string{}
	for _, m := range s.Mics() {
		ids = append(ids, m.ID)
	}
	return ids
}

// validateMicrophones synthesizes the legacy set when none is configured,
// then checks IDs, names, devices and interface channel maps.
func (s *Studio) validateMicrophones() error {
	if s.Microphones == nil {
		if len(s.FallbackMic) == 0 {
			return fmt.Errorf("studio requires microphones or fallback_mic candidates")
		}
		s.Microphones = legacyMicrophones(s.FallbackMic)
		s.legacyMicrophones = true
	} else if len(s.FallbackMic) > 0 && !s.legacyMicrophones {
		// A re-encoded legacy configuration carries the synthesized set.
		synthesized, _ := json.Marshal(legacyMicrophones(s.FallbackMic))
		configured, _ := json.Marshal(s.Microphones)
		if string(synthesized) != string(configured) {
			return fmt.Errorf("fallback_mic is replaced by device microphones; remove it")
		}
		s.legacyMicrophones = true
	}
	if len(s.Microphones) > MaxMicrophones {
		return fmt.Errorf("at most %d microphones", MaxMicrophones)
	}
	seen := map[string]bool{}
	for n := range s.Microphones {
		m := &s.Microphones[n]
		if !microphoneID.MatchString(m.ID) || m.ID == "off" || m.ID == "auto" || m.ID == "normal" || strings.HasPrefix(m.ID, "vr") {
			return fmt.Errorf("microphone id %q must be 1-24 lowercase letters, digits or dashes, and not off, auto, normal or vr…", m.ID)
		}
		if seen[m.ID] {
			return fmt.Errorf("duplicate microphone id %q", m.ID)
		}
		seen[m.ID] = true
		if m.Name == "" || len(m.Name) > 16 {
			return fmt.Errorf("microphone %s name must be 1-16 characters", m.ID)
		}
		for d := range m.Devices {
			if m.Devices[d].Driver != "wdm" {
				return fmt.Errorf("microphone %s devices must be wdm", m.ID)
			}
			if err := m.Devices[d].compile(fmt.Sprintf("microphone %s device %d", m.ID, d+1)); err != nil {
				return err
			}
		}
	}
	for n, a := range s.ASIO {
		for id, channels := range a.Inputs {
			m, _, ok := s.Microphone(id)
			if !ok || m.IsDevice() {
				return fmt.Errorf("ASIO interface %d maps %q, which is not an interface microphone", n+1, id)
			}
			if len(channels) < 1 || len(channels) > 2 || slices.ContainsFunc(channels, func(c int) bool { return c < 1 || c > 64 }) {
				return fmt.Errorf("ASIO interface %d: %s needs one channel or a left/right pair, 1-64", n+1, id)
			}
		}
	}
	return nil
}

// validMicrophoneChoice reports whether id may be chosen as a source.
func (c Config) validMicrophoneChoice(id string) bool {
	if id == "auto" || id == "off" || strings.HasPrefix(id, "vr:") {
		return true
	}
	if c.Studio == nil {
		return false
	}
	_, _, ok := c.Studio.Microphone(id)
	return ok
}
