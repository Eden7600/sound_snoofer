package streamdeck

// DefaultLayout preserves the useful Studio keys and adjacent Playback and Mic
// dials on Home, and adds Soundboard and Lights pages: a fixed frame of keys
// around a region of clips or the selected room's scenes. Home reaches both
// in one press; the page dial's press returns Home. The Apps page gives each
// app a dial and a mute key. Up and Down page through
// soundboard overflow.
func DefaultLayout() Layout {
	// Home: audio and recording on rows 1–2, strips of sessions and apps on
	// row 3 (clipped, since Home has no Up/Down), transport, Brightness and
	// Motion on the bottom row beside the go-to keys.
	home := Page{ID: "home", Name: "Home", Regions: []Region{{Source: "nowplaying.sessions", First: 18, Last: 21, Clip: true}, {Source: "appaudio.apps", First: 22, Last: 25, Clip: true}}}
	// Row 1 is the mic path then echo cancellation; row 2 is recording, left
	// to right: capture choices, Record, then tape playback.
	for n, id := range map[int]string{0: "audio.mic-mute", 1: "audio.speaker-mute", 2: "audio.monitor", 3: "audio.mode", 4: "aec.mode", 5: "aec.strength",
		9: "audio.record-mic", 10: "audio.record-computer", 11: "audio.record-tap", 12: "audio.record-toggle", 13: "audio.tape-play", 14: "audio.tape-stop", 15: "audio.tape-rew", 16: "audio.tape-ff",
		26: resetFocusID, 27: "nowplaying.prev", 28: "nowplaying.toggle", 29: "nowplaying.next", 30: "hue.brightness", 31: "hue.motion",
		33: gotoPrefix + "media", 34: gotoPrefix + "soundboard", 35: gotoPrefix + "lights"} {
		home.Keys[n] = Binding{Control: id, Label: id}
	}
	for n, id := range map[int]string{0: "audio.gain-playback", 1: "audio.gain-mic", 2: "nowplaying.dial", 3: "appaudio.focus"} {
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
	// Media and apps share a page: sessions and apps share rows 1–3 by need,
	// so a filtered category's rows go to the other; transport and the deck
	// filters sit on the bottom row. Dials are Playback, media and the focused
	// app; apps take the rest, and the media dial's while media is filtered off.
	media := Page{ID: "media", Name: "Media",
		Regions:     []Region{{Source: "nowplaying.sessions", Sources: []string{"appaudio.apps"}, First: 0, Last: 25}},
		DialRegions: []Region{{Source: "appaudio.apps", First: 0, Last: 4}}}
	for n, id := range map[int]string{17: scrollPrefix + "up", 26: scrollPrefix + "down", 27: "nowplaying.prev", 28: "nowplaying.toggle", 29: "nowplaying.next",
		30: "nowplaying.mute", 31: resetFocusID, 33: "appaudio.deck-apps", 34: "nowplaying.deck-media"} {
		media.Keys[n] = Binding{Control: id, Label: id}
	}
	for n, id := range map[int]string{0: "audio.gain-playback", 1: "nowplaying.dial", 2: "appaudio.focus"} {
		media.Dials[n] = Binding{Control: id, Label: id}
	}
	return Layout{Home: "home", Pages: []Page{home, sounds, lights, media}}
}
