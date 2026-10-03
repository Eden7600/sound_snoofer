package model

import "strings"

// InventoryDevices returns a display-only copy without recognized virtual
// endpoints. Unknown drivers remain visible; routing uses the full snapshot.
func InventoryDevices(devices []Device) []Device {
	visible := make([]Device, 0, len(devices))
	for _, device := range devices {
		if !virtualDevice(device) {
			visible = append(visible, device)
		}
	}
	return visible
}

func virtualDevice(device Device) bool {
	id := strings.ToLower(device.ID)
	for _, marker := range []string{"vbaudiovac", "vbaudiovaio", "vbaudiovm"} {
		if strings.Contains(id, marker) {
			return true
		}
	}
	name := strings.ToLower(device.Name)
	for _, marker := range []string{"voicemeeter", "vb-audio virtual", "vb-cable", "virtual audio cable", "steelseries sonar"} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}
