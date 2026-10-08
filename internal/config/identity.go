package config

import (
	"fmt"
	"regexp"
	"slices"
)

// DeviceNames is what resolution needs from the current observation: the
// current name of each identity (Windows endpoint IDs and ASIO driver
// CLSIDs), and the names available devices show.
type DeviceNames struct {
	ByID      map[string]string
	Available map[string]bool
}

// matchNone never matches a device name.
var matchNone = regexp.MustCompile(`[^\s\S]`)

func exactName(name string) *regexp.Regexp {
	return regexp.MustCompile("^" + regexp.QuoteMeta(name) + "$")
}

// identityRegex matches the device with id: its current name when active,
// else its stored label so it keeps a bus still showing that name, unless a
// different available device now has the label.
func identityRegex(id, label string, names DeviceNames) *regexp.Regexp {
	if name, ok := names.ByID[id]; ok {
		return exactName(name)
	}
	if names.Available[label] {
		return matchNone
	}
	return exactName(label)
}

// compileMatcher prepares an entry that names a device by either a pattern
// or an identity. Until resolved, an identity matches its stored label.
func compileMatcher(pattern, id, label, what string) (*regexp.Regexp, error) {
	switch {
	case id != "" && pattern != "":
		return nil, fmt.Errorf("%s has both a pattern and a device id", what)
	case id != "" && label == "":
		return nil, fmt.Errorf("%s device %s needs a name", what, id)
	case id != "":
		return exactName(label), nil
	case pattern == "":
		return nil, fmt.Errorf("%s needs a pattern or a device", what)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	return re, nil
}

func (c *Candidate) compile(what string) error {
	re, err := compileMatcher(c.Pattern, c.ID, c.Name, what)
	c.Regex = re
	return err
}

// Resolve returns a copy whose identity entries match the devices' current
// names. The saved configuration is never modified, so resolution repeats
// on every observation.
func (c Config) Resolve(names DeviceNames) Config {
	if c.Studio == nil {
		return c
	}
	studio := *c.Studio
	c.Studio = &studio
	resolve := func(list []Candidate) []Candidate {
		list = slices.Clone(list)
		for n := range list {
			if list[n].ID != "" {
				list[n].Regex = identityRegex(list[n].ID, list[n].Name, names)
			}
		}
		return list
	}
	studio.Playback = resolve(studio.Playback)
	studio.FallbackMic = resolve(studio.FallbackMic)
	if studio.Microphones != nil {
		studio.Microphones = slices.Clone(studio.Microphones)
		for n := range studio.Microphones {
			studio.Microphones[n].Devices = resolve(studio.Microphones[n].Devices)
		}
	}
	studio.ASIO = slices.Clone(studio.ASIO)
	for n := range studio.ASIO {
		a := &studio.ASIO[n]
		if a.ASIOID != "" {
			a.ASIORegex = identityRegex(a.ASIOID, a.ASIOName, names)
		}
		if a.PresenceID != "" {
			a.PresenceRegex = identityRegex(a.PresenceID, a.PresenceName, names)
		}
	}
	studio.Outputs = slices.Clone(studio.Outputs)
	for n := range studio.Outputs {
		o := &studio.Outputs[n]
		if o.DeviceID == "" {
			continue
		}
		if name, ok := names.ByID[o.DeviceID]; ok {
			o.Device = name
		} else {
			o.Inactive = true
		}
	}
	return c
}
