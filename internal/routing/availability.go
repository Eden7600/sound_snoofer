package routing

import (
	"fmt"
	"sort"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

// SelectASIO requires one physical WDM presence match and one ASIO driver.
func SelectASIO(profile *config.Studio, s model.Snapshot) (*model.Device, error) {
	d, _, err := selectInterface(profile, s)
	return d, err
}

func selectInterface(profile *config.Studio, s model.Snapshot) (*model.Device, *config.ASIOInterface, error) {
	for n := range profile.ASIO {
		a := &profile.ASIO[n]
		d, err := matchInterface(a, s)
		if err != nil {
			return nil, nil, fmt.Errorf("ASIO priority %d: %w", n+1, err)
		}
		if d != nil {
			return d, a, nil
		}
	}
	return nil, nil, nil
}

func matchInterface(profile *config.ASIOInterface, s model.Snapshot) (*model.Device, error) {
	present := 0
	for _, d := range s.Devices {
		if d.Available && d.Direction == "input" && d.Driver == "wdm" && profile.PresenceRegex.MatchString(d.Name) {
			present++
		}
	}
	if present > 1 {
		return nil, fmt.Errorf("ASIO presence pattern is ambiguous (%d WDM inputs)", present)
	}
	if present == 0 {
		return nil, nil
	}
	var asio *model.Device
	count := 0
	for _, d := range s.Devices {
		if d.Driver == "asio" && d.Direction == "output" && profile.ASIORegex.MatchString(d.Name) {
			candidate := d
			candidate.Available = true
			asio = &candidate
			count++
		}
	}
	if count != 1 {
		return nil, fmt.Errorf("connected interface requires one ASIO driver match, found %d", count)
	}
	return asio, nil
}

// MicrophoneOptions uses the same hardware evidence as routing, never ASIO
// driver installation alone. Off remains valid without a device observation.
func MicrophoneOptions(c config.Config, s model.Snapshot) []string {
	s = VRDevices(c, s)
	options := []string{}
	if c.Studio != nil && c.Studio.Voice != nil {
		if asio, a, err := selectInterface(c.Studio, s); err == nil && asio != nil {
			if a.Inputs[0] > 0 {
				options = append(options, "desk")
			}
			if a.Inputs[1] > 0 {
				options = append(options, "lav")
			}
		}
		if webcam, _ := selectDevice(c.Studio.FallbackMic, "input", s.Devices); webcam != nil {
			options = append(options, "webcam")
		}
	}
	if c.VR != nil {
		for _, h := range c.VR.Headsets {
			if d, _ := headsetDevice(c, s, "input", "vr:"+h.ID); d != nil {
				options = append(options, "vr:"+h.ID)
			}
		}
	}
	return append(options, "off")
}

// PlaybackOptions lists unique connected physical outputs within configured
// ownership. The empty choice means automatic priority selection.
func PlaybackOptions(c config.Config, s model.Snapshot) []string {
	s = VRDevices(c, s)
	s.Devices = playbackDevices(c.Studio, s)
	names := map[string]int{}
	if c.Studio != nil {
		for _, d := range model.InventoryDevices(s.Devices) {
			if !d.Available || d.Direction != "output" || (d.Driver != "wdm" && d.Driver != "asio") {
				continue
			}
			for _, candidate := range c.PlaybackCandidates() {
				if candidate.Driver == d.Driver && candidate.Regex.MatchString(d.Name) {
					names[d.Name]++
					break
				}
			}
		}
	}
	options := []string{""}
	for name, count := range names {
		if count == 1 {
			options = append(options, name)
		}
	}
	sort.Strings(options[1:])
	return options
}

// playbackDevices makes installed but unselected ASIO drivers unavailable to
// every playback priority and manual picker.
func playbackDevices(profile *config.Studio, s model.Snapshot) []model.Device {
	devices := append([]model.Device(nil), s.Devices...)
	var selected *model.Device
	if profile != nil {
		selected, _ = SelectASIO(profile, s)
	}
	for n := range devices {
		if devices[n].Driver == "asio" {
			devices[n].Available = selected != nil && devices[n].Direction == "output" && devices[n].Name == selected.Name
		}
	}
	return devices
}
