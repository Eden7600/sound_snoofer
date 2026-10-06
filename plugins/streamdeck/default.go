package streamdeck

// DefaultLayout preserves the useful Studio keys and adjacent Playback and Mic
// dials on Home, and adds Soundboard and Lights pages: a fixed frame of keys
// around a region of clips or the selected room's scenes. Home reaches both
// in one press; the page dial's press returns Home. The Apps page gives each
// app a dial and a mute key. Up and Down page through
// soundboard overflow.
func DefaultLayout() Layout {
	home := Page{ID: "home", Name: "Home"}
	for n, id := range map[int]string{0: "audio.mic-mute", 1: "audio.speaker-mute", 2: "audio.monitor", 3: "audio.mode", 8: "core.open-controls", 9: "audio.record-toggle", 11: "audio.record-mic", 12: "audio.record-computer", 13: "audio.record-tap", 27: "media.prev", 28: "media.play", 29: "media.next",
		33: gotoPrefix + "media", 34: gotoPrefix + "soundboard", 35: gotoPrefix + "lights"} {
		home.Keys[n] = Binding{Control: id, Label: id}
	}
	for n, id := range map[int]string{0: "audio.gain-playback", 1: "audio.gain-mic", 2: "nowplaying.dial"} {
		home.Dials[n] = Binding{Control: id, Label: id}
	}

	// Clips fill r1–r4 c1–c8; Overlap, Up, Down and Stop hold column 9.
	sounds := Page{ID: "soundboard", Name: "Soundboard", Regions: []Region{{Source: "soundboard.clips", First: 0, Last: 34}}}
	for n, id := range map[int]string{8: "soundboard.overlap", 17: scrollPrefix + "up", 26: scrollPrefix + "down", 35: "soundboard.stop"} {
		sounds.Keys[n] = Binding{Control: id, Label: id}
	}
	sounds.Dials[0] = Binding{Control: "soundboard.volume", Label: "soundboard.volume"}

	// Room controls on r1; the room's scenes fill r2–r4. Brightness stays on
	// dial 5 beside pagination.
	lights := Page{ID: "lights", Name: "Lights", Regions: []Region{{Source: "hue.room-scenes", First: 9, Last: 35}}}
	for n, id := range map[int]string{0: "hue.group", 1: "hue.brightness", 2: "hue.motion", 3: "hue.sync", 4: "hue.sync-mode", 5: "hue.sync-intensity"} {
		lights.Keys[n] = Binding{Control: id, Label: id}
	}
	for n, id := range map[int]string{0: "audio.gain-playback", 1: "audio.gain-mic", 4: "hue.brightness"} {
		lights.Dials[n] = Binding{Control: id, Label: id}
	}
	// Media and apps share a page: sessions on r1, app keys on r2 above their
	// dials (2–5), transport and the deck filters on the bottom row, and the
	// media dial first. App keys and dials page together.
	media := Page{ID: "media", Name: "Media",
		Regions:     []Region{{Source: "nowplaying.sessions", First: 0, Last: 7}, {Source: "appaudio.apps", First: 10, Last: 13}},
		DialRegions: []Region{{Source: "appaudio.apps", First: 1, Last: 4}}}
	for n, id := range map[int]string{17: scrollPrefix + "up", 26: scrollPrefix + "down", 27: "nowplaying.prev", 28: "nowplaying.toggle", 29: "nowplaying.next",
		30: "nowplaying.mute", 31: "nowplaying.focus", 33: "appaudio.deck-apps", 34: "nowplaying.deck-media"} {
		media.Keys[n] = Binding{Control: id, Label: id}
	}
	media.Dials[0] = Binding{Control: "nowplaying.dial", Label: "nowplaying.dial"}
	return Layout{Home: "home", Pages: []Page{home, sounds, lights, media}}
}
