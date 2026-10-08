package config

import (
	"fmt"
	"regexp"
	"slices"
)

// Output is a named listening destination with one static device, such as
// speakers for music beside headphones for monitoring. Snoofer gives the
// device its own A bus, never treats it as Playback, and owns only the sends
// of the output's sources on that bus.
type Output struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Device string `json:"device"` // Exact WDM output name.
	// Sources are the defaults for the saved per-source switches.
	Sources []string `json:"sources,omitempty"`
}

// MaxOutputs bounds slots by the A buses left beside ASIO and Playback.
const MaxOutputs = 3

// Output sources besides the configured playback sources.
const (
	SourceMonitor    = "monitor"
	SourceSoundboard = "soundboard"
	SourceTape       = "tape"
)

var outputID = regexp.MustCompile(`^[a-z0-9-]{1,24}$`)

// OutputSources lists the sources an output can receive, in display order.
func (s *Studio) OutputSources() []string {
	sources := append(slices.Clone(s.PlaybackSources), SourceMonitor, SourceSoundboard)
	if s.Recording != nil {
		sources = append(sources, SourceTape)
	}
	return sources
}

// OutputDevice reports whether name is an output slot's device.
func (s *Studio) OutputDevice(name string) bool {
	return name != "" && slices.ContainsFunc(s.Outputs, func(o Output) bool { return o.Device == name })
}

func (s *Studio) validateOutputs() error {
	if len(s.Outputs) > MaxOutputs {
		return fmt.Errorf("at most %d outputs", MaxOutputs)
	}
	ids, devices := map[string]bool{}, map[string]bool{}
	for _, o := range s.Outputs {
		if !outputID.MatchString(o.ID) {
			return fmt.Errorf("output id %q must be 1-24 lowercase letters, digits or dashes", o.ID)
		}
		if ids[o.ID] {
			return fmt.Errorf("duplicate output id %q", o.ID)
		}
		ids[o.ID] = true
		if o.Name == "" || len(o.Name) > 16 {
			return fmt.Errorf("output %s name must be 1-16 characters", o.ID)
		}
		if o.Device == "" {
			return fmt.Errorf("output %s needs a device", o.ID)
		}
		if devices[o.Device] {
			return fmt.Errorf("device %q is used by two outputs", o.Device)
		}
		devices[o.Device] = true
		seen := map[string]bool{}
		for _, source := range o.Sources {
			if !slices.Contains(s.OutputSources(), source) || seen[source] {
				return fmt.Errorf("output %s has invalid or duplicate source %q", o.ID, source)
			}
			seen[source] = true
		}
	}
	return nil
}

// NormalizeOutputs gives every configured output a switch per source, using
// its configured defaults where nothing is saved, and drops switches for
// removed outputs or sources. Adding an output never invalidates choices.
func (i *Intent) NormalizeOutputs(c Config) {
	if c.Studio == nil || len(c.Studio.Outputs) == 0 {
		i.Outputs = nil
		return
	}
	next := map[string]map[string]bool{}
	for _, o := range c.Studio.Outputs {
		switches := map[string]bool{}
		for _, source := range c.Studio.OutputSources() {
			on, saved := i.Outputs[o.ID][source]
			if !saved {
				on = slices.Contains(o.Sources, source)
			}
			switches[source] = on
		}
		next[o.ID] = switches
	}
	i.Outputs = next
}

// OutputOn reports whether output id receives source.
func (i *Intent) OutputOn(id, source string) bool {
	return i != nil && i.Outputs[id][source]
}
