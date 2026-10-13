package streamdeck

import "fmt"

// DefaultLayout preserves the useful Studio keys and adjacent Playback and Mic
// dials on Home, and adds Soundboard and Lights pages: a fixed frame of keys
// around a region of clips or the selected room's scenes. Home reaches both
// in one press, as it does Media and Meetings; the page dial's press returns
// Home. Routing switches sources per destination. The Apps page gives each
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
		8: gotoPrefix + "routing",
		9: "audio.record-mic", 10: "audio.record-computer", 11: "audio.record-tap", 12: "audio.record-toggle", 13: "audio.tape-play", 14: "audio.tape-stop", 15: "audio.tape-rew", 16: "audio.tape-ff",
		26: resetFocusID, 27: "nowplaying.prev", 28: "nowplaying.toggle", 29: "nowplaying.next", 30: "hue.brightness", 31: "hue.motion",
		32: gotoPrefix + "meetings", 33: gotoPrefix + "media", 34: gotoPrefix + "soundboard", 35: gotoPrefix + "lights"} {
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

	// Room controls and Sort on r1; the room's scenes fill r2–r4. Dials: Playback,
	// Saturation, Decor, Mixed, and Brightness on dial 5 beside pagination.
	lights := Page{ID: "lights", Name: "Lights", Regions: []Region{{Source: "hue.room-scenes", First: 9, Last: 35}}}
	for n, id := range map[int]string{0: "hue.group", 1: "hue.brightness", 2: "hue.motion", 3: "hue.sort", 4: "hue.sync", 5: "hue.sync-mode", 6: "hue.sync-intensity"} {
		lights.Keys[n] = Binding{Control: id, Label: id}
	}
	for n, id := range map[int]string{0: "audio.gain-playback", 1: "hue.saturation", 2: "hue.decor", 3: "hue.mixed", 4: "hue.brightness"} {
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
	// Meetings: the call on row 1, with Leave apart at c9 so it is not pressed
	// by accident beside Mute.
	meetings := Page{ID: "meetings", Name: "Meetings"}
	for n, id := range map[int]string{0: "discord.mute", 1: "discord.deafen", 2: "discord.video", 3: "discord.screenshare", 8: "discord.leave"} {
		meetings.Keys[n] = Binding{Control: id, Label: id}
	}
	for n, id := range map[int]string{0: "audio.gain-playback", 1: "audio.gain-mic"} {
		meetings.Dials[n] = Binding{Control: id, Label: id}
	}
	// Routing: destinations by row (Playback, then output slots 1–3) and
	// sources by column (Computer, Monitor, Soundboard, Tape). Positions
	// without a slot or source are blank keys.
	routing := Page{ID: "routing", Name: "Routing"}
	for n, id := range []string{"audio.playback-device", "audio.playback:virtual:1", "audio.monitor"} {
		routing.Keys[n] = Binding{Control: id, Label: id}
	}
	for slot := 1; slot <= 3; slot++ {
		for column, suffix := range []string{"", ":virtual:1", ":monitor", ":soundboard", ":tape"} {
			id := fmt.Sprintf("audio.slot-%d%s", slot, suffix)
			routing.Keys[slot*9+column] = Binding{Control: id, Label: id}
		}
	}
	for n, id := range map[int]string{0: "audio.gain-playback", 1: "audio.gain-mic"} {
		routing.Dials[n] = Binding{Control: id, Label: id}
	}
	return Layout{Home: "home", Pages: []Page{home, sounds, lights, media, meetings, routing}}
}

// DefaultLayoutFor is a model's shipped layout. Every model has the same
// page IDs, so mirrored paging lines up, and keeps the + XL's frame scaled to
// its grid: fixed controls on an edge, content in regions, Playback and Mic
// on dials 1–2 (openspec/changes/multi-model-stream-deck).
func DefaultLayoutFor(model string) Layout {
	switch model {
	case "plus":
		return smallLayout(true).forModel(model)
	case "neo":
		return smallLayout(false).forModel(model)
	case "mk2":
		return mk2Layout().forModel(model)
	case "xl":
		return xlLayout().forModel(model)
	}
	return DefaultLayout()
}

// defaultPage builds a page from key and dial bindings by position.
func defaultPage(id, name string, keys map[int]string, dials map[int]string, regions ...Region) Page {
	p := Page{ID: id, Name: name, Regions: regions}
	for n, control := range keys {
		p.Keys[n] = Binding{Control: control, Label: control}
	}
	for n, control := range dials {
		p.Dials[n] = Binding{Control: control, Label: control}
	}
	return p
}

// smallLayout is the 4 × 2 layout of the + (with its three user dials) and
// the Neo (without).
func smallLayout(dials bool) Layout {
	pick := func(m map[int]string) map[int]string {
		if !dials {
			return nil
		}
		return m
	}
	home := defaultPage("home", "Home", map[int]string{0: "audio.mic-mute", 1: "audio.speaker-mute", 2: "audio.monitor", 3: "audio.record-toggle",
		4: gotoPrefix + "media", 5: gotoPrefix + "soundboard", 6: gotoPrefix + "lights", 7: gotoPrefix + "meetings"},
		pick(map[int]string{0: "audio.gain-playback", 1: "audio.gain-mic", 2: "nowplaying.dial"}))
	sounds := defaultPage("soundboard", "Soundboard", map[int]string{3: "soundboard.stop", 7: scrollPrefix + "down"},
		pick(map[int]string{0: "soundboard.volume"}), Region{Source: "soundboard.clips", First: 0, Last: 6})
	lights := defaultPage("lights", "Lights", map[int]string{0: "hue.group", 1: "hue.brightness", 2: "hue.motion", 3: "hue.sort"},
		pick(map[int]string{0: "audio.gain-playback", 1: "hue.saturation", 2: "hue.brightness"}), Region{Source: "hue.room-scenes", First: 4, Last: 7})
	media := defaultPage("media", "Media", map[int]string{3: scrollPrefix + "up", 7: scrollPrefix + "down"},
		pick(map[int]string{0: "audio.gain-playback", 1: "nowplaying.dial", 2: "appaudio.focus"}),
		Region{Source: "nowplaying.sessions", Sources: []string{"appaudio.apps"}, First: 0, Last: 6})
	meetings := defaultPage("meetings", "Meetings", map[int]string{0: "discord.mute", 1: "discord.deafen", 2: "discord.video", 3: "discord.leave"},
		pick(map[int]string{0: "audio.gain-playback", 1: "audio.gain-mic"}))
	return Layout{Home: "home", Pages: []Page{home, sounds, lights, media, meetings}}
}

// mk2Layout is the 5 × 3 layout of the MK.2 and Original; key 15 pages.
func mk2Layout() Layout {
	home := defaultPage("home", "Home", map[int]string{0: "audio.mic-mute", 1: "audio.speaker-mute", 2: "audio.monitor", 3: "audio.mode", 4: "audio.record-toggle",
		5: "nowplaying.prev", 6: "nowplaying.toggle", 7: "nowplaying.next", 8: "hue.brightness", 9: "hue.motion",
		10: gotoPrefix + "meetings", 11: gotoPrefix + "media", 12: gotoPrefix + "soundboard", 13: gotoPrefix + "lights"}, nil)
	sounds := defaultPage("soundboard", "Soundboard", map[int]string{4: "soundboard.stop", 9: scrollPrefix + "down"}, nil,
		Region{Source: "soundboard.clips", First: 0, Last: 13})
	lights := defaultPage("lights", "Lights", map[int]string{0: "hue.group", 1: "hue.brightness", 2: "hue.motion", 3: "hue.sort", 4: "hue.sync"}, nil,
		Region{Source: "hue.room-scenes", First: 5, Last: 14})
	media := defaultPage("media", "Media", map[int]string{4: scrollPrefix + "up", 9: scrollPrefix + "down",
		10: "nowplaying.prev", 11: "nowplaying.toggle", 12: "nowplaying.next", 13: "nowplaying.mute"}, nil,
		Region{Source: "nowplaying.sessions", Sources: []string{"appaudio.apps"}, First: 0, Last: 8})
	meetings := defaultPage("meetings", "Meetings", map[int]string{0: "discord.mute", 1: "discord.deafen", 2: "discord.video", 3: "discord.screenshare", 4: "discord.leave"}, nil)
	return Layout{Home: "home", Pages: []Page{home, sounds, lights, media, meetings}}
}

// xlLayout is the + XL layout on the XL's 8 × 4 grid: without column 9 and
// the dial-only keys; key 32 pages.
func xlLayout() Layout {
	home := defaultPage("home", "Home", map[int]string{0: "audio.mic-mute", 1: "audio.speaker-mute", 2: "audio.monitor", 3: "audio.mode", 4: "aec.mode", 5: "aec.strength",
		7: gotoPrefix + "routing",
		8: "audio.record-mic", 9: "audio.record-computer", 10: "audio.record-tap", 11: "audio.record-toggle", 12: "audio.tape-play", 13: "audio.tape-stop", 14: "audio.tape-rew", 15: "audio.tape-ff",
		24: "nowplaying.prev", 25: "nowplaying.toggle", 26: "nowplaying.next",
		27: gotoPrefix + "meetings", 28: gotoPrefix + "media", 29: gotoPrefix + "soundboard", 30: gotoPrefix + "lights"}, nil,
		Region{Source: "nowplaying.sessions", First: 16, Last: 19, Clip: true}, Region{Source: "appaudio.apps", First: 20, Last: 23, Clip: true})
	sounds := defaultPage("soundboard", "Soundboard", map[int]string{7: "soundboard.stop", 15: "soundboard.overlap", 23: scrollPrefix + "down"}, nil,
		Region{Source: "soundboard.clips", First: 0, Last: 30})
	lights := defaultPage("lights", "Lights", map[int]string{0: "hue.group", 1: "hue.brightness", 2: "hue.motion", 3: "hue.sort", 4: "hue.sync", 5: "hue.sync-mode", 6: "hue.sync-intensity"}, nil,
		Region{Source: "hue.room-scenes", First: 8, Last: 31})
	media := defaultPage("media", "Media", map[int]string{15: scrollPrefix + "up", 23: scrollPrefix + "down",
		24: "nowplaying.prev", 25: "nowplaying.toggle", 26: "nowplaying.next", 27: "nowplaying.mute", 29: "appaudio.deck-apps", 30: "nowplaying.deck-media"}, nil,
		Region{Source: "nowplaying.sessions", Sources: []string{"appaudio.apps"}, First: 0, Last: 22})
	meetings := defaultPage("meetings", "Meetings", map[int]string{0: "discord.mute", 1: "discord.deafen", 2: "discord.video", 3: "discord.screenshare", 7: "discord.leave"}, nil)
	routing := defaultPage("routing", "Routing", map[int]string{0: "audio.playback-device", 1: "audio.playback:virtual:1", 2: "audio.monitor"}, nil)
	for slot := 1; slot <= 3; slot++ {
		for column, suffix := range []string{"", ":virtual:1", ":monitor", ":soundboard", ":tape"} {
			id := fmt.Sprintf("audio.slot-%d%s", slot, suffix)
			routing.Keys[slot*8+column] = Binding{Control: id, Label: id}
		}
	}
	return Layout{Home: "home", Pages: []Page{home, sounds, lights, media, meetings, routing}}
}
