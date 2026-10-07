package audio

import "sound-snoofer/internal/model"

type tapeRow struct {
	id, label, value, icon string
	available              bool
}

// tapeRows are the recorder playback controls. None can act on a recording:
// all are unavailable while recording or while the recorder state is unknown,
// and Stop only acts on playback.
func tapeRows(r *model.RecorderSnapshot) []tapeRow {
	state := r.State()
	playing := r.TapePlaying()
	idle := state == "Stopped" || playing
	play := tapeRow{id: "tape-play", label: "Play recording", value: "Ready", icon: "tape-play", available: idle}
	switch {
	case playing && state == "Playing":
		play.value, play.icon = "Playing", "tape-pause"
	case playing:
		play.value = "Paused"
	case state == "Recording" || state == "Paused":
		play.value = "Rec"
	case state == "Unknown":
		play.value = "N/A"
	}
	return []tapeRow{
		play,
		{id: "tape-stop", label: "Stop playback", icon: "tape-stop", available: playing},
		{id: "tape-rew", label: "Rewind", icon: "tape-rew", available: idle},
		{id: "tape-ff", label: "Fast-forward", icon: "tape-ff", available: idle},
	}
}
