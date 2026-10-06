package streamdeck

// DefaultLayout preserves the useful Studio keys, Playback dial and microphone position.
func DefaultLayout() Layout {
	p := Page{ID: "home", Name: "Home"}
	for n, id := range map[int]string{0: "audio.mic-mute", 1: "audio.speaker-mute", 2: "audio.monitor", 3: "audio.mode", 8: "core.open-controls", 9: "audio.record-toggle", 11: "audio.record-mic", 12: "audio.record-computer", 13: "audio.record-tap", 27: "media.prev", 28: "media.play", 29: "media.next"} {
		p.Keys[n] = Binding{Control: id, Label: id}
	}
	for n, id := range map[int]string{0: "audio.gain-playback", 2: "audio.gain-mic"} {
		p.Dials[n] = Binding{Control: id, Label: id}
	}
	return Layout{Home: "home", Pages: []Page{p}}
}
