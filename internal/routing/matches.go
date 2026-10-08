package routing

import (
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

// candidateMatch is the planner's rule for a candidate matching a device.
func candidateMatch(c config.Candidate, direction string, d model.Device) bool {
	return d.Available && d.Driver == c.Driver && d.Direction == direction && c.Regex.MatchString(d.Name)
}

// candidateMatches lists the available devices c matches; a candidate is
// usable only with exactly one match.
func candidateMatches(c config.Candidate, direction string, devices []model.Device) []string {
	names := []string{}
	for _, d := range devices {
		if candidateMatch(c, direction, d) {
			names = append(names, d.Name)
		}
	}
	return names
}

// PlaybackMatches lists the devices a playback candidate matches under the
// planner's rules: only the selected interface's ASIO driver is available.
func PlaybackMatches(profile *config.Studio, c config.Candidate, s model.Snapshot) []string {
	return candidateMatches(c, "output", playbackDevices(profile, s))
}

// WebcamMatches lists the devices a fallback microphone candidate matches.
func WebcamMatches(c config.Candidate, s model.Snapshot) []string {
	return candidateMatches(c, "input", s.Devices)
}

// InterfaceMatches lists the available WDM inputs an interface's presence
// pattern matches and the installed ASIO drivers its driver pattern matches.
// Drivers alone never prove the interface is present.
func InterfaceMatches(a config.ASIOInterface, s model.Snapshot) (presence, drivers []string) {
	presence, drivers = []string{}, []string{}
	for _, d := range s.Devices {
		if d.Available && d.Direction == "input" && d.Driver == "wdm" && a.PresenceRegex.MatchString(d.Name) {
			presence = append(presence, d.Name)
		}
		if d.Driver == "asio" && d.Direction == "output" && a.ASIORegex.MatchString(d.Name) {
			drivers = append(drivers, d.Name)
		}
	}
	return presence, drivers
}
