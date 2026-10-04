package routing

import (
	"fmt"
	"sort"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

func selectASIO(profile *config.Studio, s model.Snapshot) (*model.Device, error) {
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
	options := []string{}
	if c.Studio != nil && c.Studio.Voice != nil {
		if asio, err := selectASIO(c.Studio, s); err == nil && asio != nil {
			options = append(options, "desk", "lav")
		}
		if webcam, _ := selectDevice(c.Studio.FallbackMic, "input", s.Devices); webcam != nil {
			options = append(options, "webcam")
		}
	}
	return append(options, "off")
}

// PlaybackOptions lists unique connected physical outputs within configured
// ownership. The empty choice means automatic priority selection.
func PlaybackOptions(c config.Config, s model.Snapshot) []string {
	names := map[string]int{}
	if c.Studio != nil {
		for _, d := range model.InventoryDevices(s.Devices) {
			if !d.Available || d.Direction != "output" || d.Driver != "wdm" {
				continue
			}
			for _, candidate := range c.Studio.Playback {
				if candidate.Regex.MatchString(d.Name) {
					names[d.Name]++
					break
				}
			}
		}
	}
	if c.Studio != nil && c.Studio.ASIOPlayback {
		if asio, err := selectASIO(c.Studio, s); err == nil && asio != nil {
			names[asio.Name]++
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
