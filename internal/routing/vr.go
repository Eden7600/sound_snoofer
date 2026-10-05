package routing

import (
	"fmt"
	"regexp"
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

func vrRunning(s model.Snapshot) bool {
	return s.SteamVR != nil && s.SteamVR.Known && s.SteamVR.Running
}
func vrRegex(h config.Headset, direction string) *regexp.Regexp {
	if direction == "input" {
		return h.MicRegex
	}
	return h.PlaybackRegex
}

// VRDevices gates classified endpoints before any generic matcher can select them.
func VRDevices(c config.Config, s model.Snapshot) model.Snapshot {
	if c.VR == nil {
		return s
	}
	devices := append([]model.Device(nil), s.Devices...)
	for n, d := range devices {
		owners := 0
		var pattern *regexp.Regexp
		for _, h := range c.VR.Headsets {
			re := vrRegex(h, d.Direction)
			if re != nil && re.MatchString(d.Name) {
				owners++
				pattern = re
			}
		}
		if owners == 0 {
			continue
		}
		matches := 0
		for _, other := range s.Devices {
			if other.Available && other.Driver == "wdm" && other.Direction == d.Direction && pattern.MatchString(other.Name) {
				matches++
			}
		}
		devices[n].Available = d.Available && d.Driver == "wdm" && vrRunning(s) && owners == 1 && matches == 1
	}
	s.Devices = devices
	return s
}
func headsetDevice(c config.Config, s model.Snapshot, direction, id string) (*model.Device, string) {
	if c.VR == nil {
		return nil, ""
	}
	s = VRDevices(c, s)
	for _, h := range c.VR.Headsets {
		if id != "" && id != "vr:"+h.ID {
			continue
		}
		re := vrRegex(h, direction)
		if re == nil {
			continue
		}
		for _, d := range s.Devices {
			if d.Available && d.Driver == "wdm" && d.Direction == direction && re.MatchString(d.Name) {
				copy := d
				return &copy, "vr:" + h.ID
			}
		}
	}
	return nil, ""
}
func vrPlaybackCandidates(c config.Config) []config.Candidate {
	out := append([]config.Candidate(nil), c.Studio.Playback...)
	if c.VR != nil {
		for _, h := range c.VR.Headsets {
			if h.PlaybackRegex != nil {
				out = append(out, config.Candidate{Driver: "wdm", Pattern: h.Playback, Regex: h.PlaybackRegex})
			}
		}
	}
	return out
}
func managedMicStrips(c config.Config) []int {
	out := []int{0, 1, 2, 6}
	if c.VR != nil {
		out = append(out, c.VR.Input-1)
	}
	return out
}
func addVRMic(c config.Config, s model.Snapshot, t *Topology, source int) (int, string, error) {
	if c.VR == nil {
		return source, "", nil
	}
	i := c.VoiceIntent()
	slot := fmt.Sprintf("input:%d", c.VR.Input)
	current := s.Assignments[slot]
	owned := current == ""
	for _, h := range c.VR.Headsets {
		if h.MicRegex != nil && h.MicRegex.MatchString(current) {
			owned = true
		}
	}
	if !owned {
		return source, "", fmt.Errorf("%s occupied by unmanaged microphone", slot)
	}
	var d *model.Device
	id := ""
	if i.MicActive() {
		if i.PreferVRMic {
			d, id = headsetDevice(c, s, "input", "")
		}
		if d == nil {
			d, id = headsetDevice(c, s, "input", i.Source)
			if len(i.Source) < 3 || i.Source[:3] != "vr:" {
				d = nil
				id = ""
			}
		}
	}
	if d == nil {
		d = &model.Device{Direction: "input", Driver: "wdm", Available: true}
	} else {
		source = c.VR.Input - 1
	}
	t.Operations = append(t.Operations, Operation{Target: slot, Device: d, BeforeName: current, Change: current != d.Name})
	return source, id, nil
}
