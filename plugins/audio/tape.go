package audio

import "sound-snoofer/internal/model"

type tapeRow struct {
	id, label, value, icon string
	available              bool
}

// recordingActive reports a recording in progress or paused, as opposed to
// paused playback.
func recordingActive(r *model.RecorderSnapshot) bool {
	state := r.State()
	return state == "Recording" || (state == "Paused" && !r.TapePlaying())
}

// tapeRows are the tape-deck transport keys (openspec/changes/recorder-transport).
// Play/Pause pauses the tape or the recording; Stop returns the tape to the
// start or ends the recording; Rewind and Fast-forward never act on a
// recording. paused is Snoofer's knowledge that its stop paused playback.
func tapeRows(r *model.RecorderSnapshot, paused bool) []tapeRow {
	state := r.State()
	known := state != "Unknown"
	recording := recordingActive(r)
	playback := r.TapePlaying()
	play := tapeRow{id: "tape-play", label: "Play Recording", value: "Ready", icon: "tape-play", available: known}
	switch {
	case !known:
		play.value = "N/A"
	case state == "Recording":
		play.value, play.icon = "Rec", "tape-pause"
	case recording:
		play.value = "Paused"
	case state == "Playing":
		play.value, play.icon = "Playing", "tape-pause"
	case playback || paused:
		play.value = "Paused"
	}
	// Stop, Rewind and Fast-forward read Ready; surfaces show N/A while unavailable.
	return []tapeRow{
		play,
		{id: "tape-stop", label: "Stop", value: "Ready", icon: "tape-stop", available: known && (recording || playback || paused)},
		{id: "tape-rew", label: "Rewind", value: "Ready", icon: "tape-rew", available: known && !recording},
		{id: "tape-ff", label: "Fast-forward", value: "Ready", icon: "tape-ff", available: known && !recording},
	}
}
