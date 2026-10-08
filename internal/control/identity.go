package control

import (
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
	"sound-snoofer/internal/windowsaudio"
)

// DeviceNames maps device identities to current names: Windows endpoint IDs
// from the endpoint inventory and ASIO driver CLSIDs from Voicemeeter's
// inventory. It also lists the names available devices show.
func DeviceNames(endpoints []windowsaudio.Endpoint, s model.Snapshot) config.DeviceNames {
	names := config.DeviceNames{ByID: map[string]string{}, Available: map[string]bool{}}
	for _, e := range endpoints {
		names.ByID[e.ID] = e.Name
	}
	for _, d := range s.Devices {
		if d.Driver == "asio" && d.ID != "" {
			names.ByID[d.ID] = d.Name
		}
		if d.Available {
			names.Available[d.Name] = true
		}
	}
	return names
}
