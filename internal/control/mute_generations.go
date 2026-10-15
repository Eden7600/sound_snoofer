package control

import "sound-snoofer/internal/config"

// These generations belong only to the two global preferences, not physical targets.
type muteGenerations struct {
	mic, playback                    uint64
	present, micMuted, playbackMuted bool
}

func newMuteGenerations(intent *config.Intent) muteGenerations {
	g := muteGenerations{mic: 1, playback: 1, present: intent != nil}
	if intent != nil {
		g.micMuted, g.playbackMuted = intent.MicMuted, intent.PlaybackMuted
	}
	return g
}

func (g *muteGenerations) update(intent *config.Intent, invalidate bool) {
	present := intent != nil
	micMuted, playbackMuted := false, false
	if present {
		micMuted, playbackMuted = intent.MicMuted, intent.PlaybackMuted
	}
	if invalidate || present != g.present {
		g.mic++
		g.playback++
	} else {
		if micMuted != g.micMuted {
			g.mic++
		}
		if playbackMuted != g.playbackMuted {
			g.playback++
		}
	}
	g.present, g.micMuted, g.playbackMuted = present, micMuted, playbackMuted
}

func (g *muteGenerations) matches(a Action) bool {
	if !g.present || a.Kind != editRule || a.MuteGeneration == 0 {
		return false
	}
	switch a.Row {
	case "mic-mute":
		return a.MuteGeneration == g.mic
	case "speaker-mute":
		return a.MuteGeneration == g.playback
	}
	return false
}
