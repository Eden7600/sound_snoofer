package audio

import "strings"

// shortLabel retains only context not already carried by the key's icon.
func shortLabel(key string) string {
	if strings.HasPrefix(key, "playback:virtual:") {
		return "Playback " + strings.TrimPrefix(key, "playback:virtual:")
	}
	prefix := ""
	if strings.HasPrefix(key, "normal-") {
		prefix, key = "Normal ", strings.TrimPrefix(key, "normal-")
	} else if strings.HasPrefix(key, "vr-profile-") {
		prefix, key = "VR ", strings.TrimPrefix(key, "vr-profile-")
	}
	label := map[string]string{
		"source": "Mic target", "mode": "Mic processing", "monitor": "Monitor", "output": "Playback",
		"mic-stack": "Mic stack", "mic-mute": "Mute", "speaker-mute": "Mute",
		"defaults": "Defaults", "auto-recover": "Recovery",
		"record-mic": "Record mic", "record-computer": "Record PC", "record-loop": "Loop",
		"record-vst": "To VST", "record-tap": "Mic stage", "record-start": "Record",
		"record-stop": "Stop rec", "record-toggle": "Record", "snippet-play": "Play", "snippet-stop": "Stop",
		"gain-playback": "Playback", "gain-mic": "Mic", "engine-restart": "Restart", "engine-confirm": "Confirm restart",
	}[key]
	if label == "" {
		return ""
	}
	// Fixed-profile bindings must stay distinct when placed beside active controls.
	if prefix != "" {
		label = strings.TrimPrefix(label, "Mic ")
		if key == "mode" {
			label = "FX"
		}
	}
	return prefix + label
}
