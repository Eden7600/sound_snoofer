package routing

import (
	"slices"
	"strings"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

// ProfileConfig resolves an immutable saved configuration for each fresh snapshot.
// ProfileBase prevents derived choices from overwriting saved Normal preferences.
// Runtime-only state from the caller is kept across that restore.
func ProfileConfig(c config.Config, s model.Snapshot) config.Config {
	if c.ProfileBase != nil {
		tapeListening, silent := c.TapeListening, c.SilentMics
		c = *c.ProfileBase
		c.TapeListening, c.SilentMics = tapeListening, silent
	}
	if c.Profiles == nil || c.Studio == nil || c.VoiceIntent() == nil {
		return c
	}
	base := c
	i := c.VoiceIntent()
	stackEnabled := i.Enabled
	profile := *c.Studio
	c.Studio = &profile
	policy := (*config.ProfilePolicy)(nil)
	if c.Policy != nil {
		policy = c.Policy()
	}

	if policy != nil {
		devices := policy.Devices
		c.VR = &devices
		c.ProfileRunning = policy.Running
	}
	available := s
	available.SteamVR = &model.ProcessStatus{Known: true, Running: c.ProfileRunning}
	available = VRDevices(c, available)
	options := MicrophoneOptions(c, available)
	normalSource := i.Source
	normalMissing := false
	if normalSource == "auto" || !slices.Contains(options, normalSource) {
		normalSource = "off"
		normalMissing = true
		// With activity metering, Auto skips silent microphones; when every
		// option is silent, the priority applies regardless.
		for _, skipSilent := range []bool{c.Profiles.Activity != nil, false} {
			for _, id := range c.Profiles.Microphones {
				if slices.Contains(options, id) && !(skipSilent && slices.Contains(c.SilentMics, id)) {
					normalSource = id
					normalMissing = false
					break
				}
			}
			if !normalMissing {
				break
			}
		}
	}
	c.ProfileWired = wiredMics(c.Profiles, options, normalSource)
	normalPlayback, _ := selectDevice(profile.Playback, "output", playbackDevices(&profile, available))
	asio, _ := SelectASIO(&profile, available)
	chooseOverride := func(name string, current *model.Device, choices config.Config) *model.Device {
		if name == "" || !slices.Contains(PlaybackOptions(choices, available), name) {
			return current
		}
		if asio != nil && asio.Name == name {
			return asio
		}
		matches := []model.Device{}
		for _, d := range available.Devices {
			if d.Available && d.Direction == "output" && d.Driver == "wdm" && d.Name == name {
				matches = append(matches, d)
			}
		}
		if len(matches) == 1 {
			d := matches[0]
			return &d
		}
		return current
	}
	normalPlayback = chooseOverride(i.PlaybackDevice, normalPlayback, c)
	c.ProfilePlayback = normalPlayback
	c.ProfileMicMissing = normalMissing
	i.Source = normalSource
	i.Enabled = normalSource != "off"
	if c.ProfileRunning && policy != nil {
		// The VR profile keeps every microphone wired.
		c.ProfileWired = nil
		choices := policy.Choices
		if i.VRProfile != nil {
			choices = *i.VRProfile
		}
		c.ProfileMicMissing = false
		i.PlaybackDevice = choices.Playback
		i.Mode = choices.Mode
		i.Monitor = choices.Monitor
		source := ""
		if choices.Source == "off" {
			source = "off"
		} else if choices.Source != "auto" && slices.Contains(options, choices.Source) {
			source = choices.Source
		}
		if source == "" {
			for _, id := range policy.Microphones {
				if id == "normal" {
					source = normalSource
					c.ProfileMicMissing = normalMissing
					break
				}
				if slices.Contains(options, id) {
					source = id
					break
				}
			}
		}
		if source == "" {
			source = "off"
			c.ProfileMicMissing = true
		}
		i.Source = source
		i.Enabled = source != "off"
		c.ProfilePlayback = nil
		for _, candidate := range policy.Playback {
			if candidate.Driver == "normal" {
				c.ProfilePlayback = normalPlayback
				break
			}
			d, _ := selectDevice([]config.Candidate{candidate}, "output", playbackDevices(&profile, available))
			if d != nil {
				c.ProfilePlayback = d
				break
			}
		}
		overrideConfig := c
		profileCopy := profile
		profileCopy.Playback = slices.Clone(profile.Playback)
		for _, candidate := range policy.Playback {
			if candidate.Driver != "normal" {
				profileCopy.Playback = append(profileCopy.Playback, candidate)
			}
		}
		overrideConfig.Studio = &profileCopy
		c.ProfilePlayback = chooseOverride(choices.Playback, c.ProfilePlayback, overrideConfig)
	}
	// Retain ownership in Normal too, without adding VR devices to its priorities.
	owned := c.PolicyPlayback
	if policy != nil {
		owned = policy.Playback
	}
	for _, candidate := range owned {
		if candidate.Driver != "normal" {
			profile.Playback = append(slices.Clone(profile.Playback), candidate)
		}
	}
	if i.Source == "auto" {
		i.Source = "off"
		i.Enabled = false
	}
	if strings.HasPrefix(i.Source, "vr:") && !slices.Contains(options, i.Source) {
		i.Source = "off"
		i.Enabled = false
		c.ProfileMicMissing = true
	}
	i.Enabled = stackEnabled && i.Enabled
	if !stackEnabled {
		c.ProfileMicMissing = false
	}
	i.PreferVRMic = false
	i.PreferVRPlayback = false
	c.Intent = i
	c.ProfileResolved = true
	c.ProfileBase = &base
	return c
}

// wiredMics lists the microphones wired for activity metering: the first
// Check available options in priority order and the effective source. Nil,
// without activity metering, wires every available microphone.
func wiredMics(p *config.Profiles, options []string, effective string) []string {
	if p.Activity == nil {
		return nil
	}
	wired := []string{}
	for _, id := range p.Microphones {
		if len(wired) == p.Activity.Check {
			break
		}
		if id != "off" && slices.Contains(options, id) {
			wired = append(wired, id)
		}
	}
	if effective != "off" && !slices.Contains(wired, effective) {
		wired = append(wired, effective)
	}
	return wired
}
