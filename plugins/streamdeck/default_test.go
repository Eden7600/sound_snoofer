package streamdeck

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"sound-snoofer/snoofer"
)

func TestDefaultLayoutPages(t *testing.T) {
	l := DefaultLayout()
	if err := l.Validate(nil); err != nil {
		t.Fatal(err)
	}
	if len(l.Pages) != 4 || l.Pages[0].ID != "home" || l.Pages[1].ID != "soundboard" || l.Pages[2].ID != "lights" || l.Pages[3].ID != "media" {
		t.Fatal("default pages", l.Pages)
	}
	for n, id := range []string{resetFocusID, "nowplaying.prev", "nowplaying.toggle", "nowplaying.next"} {
		if l.Pages[0].Keys[26+n].Control != id {
			t.Fatal("Home transport", l.Pages[0].Keys[27:30])
		}
	}
	if l.Pages[0].Keys[35].Control != gotoPrefix+"lights" || l.Pages[0].Keys[34].Control != gotoPrefix+"soundboard" || l.Pages[0].Keys[33].Control != gotoPrefix+"media" || l.Pages[0].Keys[32].Control != "" {
		t.Fatal("Home lacks one-step reach")
	}
	var controls []snoofer.Control
	for n := 0; n < 40; n++ {
		controls = append(controls, snoofer.Control{ID: fmt.Sprintf("soundboard.clip-%02d", n), Label: fmt.Sprintf("Clip %02d", n), Collection: "soundboard.clips", Operations: []string{"press"}})
	}
	for n := 1; n <= 3; n++ {
		controls = append(controls, snoofer.Control{ID: fmt.Sprintf("hue.room-scene-%d", n), Label: fmt.Sprintf("Scene %d", n), Collection: "hue.room-scenes", Operations: []string{"press"}})
	}
	expanded := l.expanded(controls)
	// 31 clip cells: 40 clips overflow to a second Soundboard set, which the
	// page dial skips because the page binds scroll keys.
	if len(expanded.Pages) != 5 || expanded.Pages[2].ID != "soundboard~auto~2" {
		t.Fatal("expanded pages", len(expanded.Pages))
	}
	if expanded.next("soundboard", 1) != "lights" || expanded.next("soundboard~auto~2", -1) != "home" || expanded.next("lights", -1) != "soundboard" {
		t.Fatal("page dial visits overflow sets")
	}
	for _, p := range expanded.Pages[1:3] {
		if p.Keys[35].Control != "soundboard.stop" || p.Keys[8].Control != "soundboard.overlap" || p.Keys[17].Control != scrollPrefix+"up" || p.Keys[26].Control != scrollPrefix+"down" {
			t.Fatal("soundboard frame moved on", p.ID)
		}
	}
	lights := expanded.Pages[3]
	if lights.Keys[9].Control != "hue.room-scene-1" || lights.Keys[0].Control != "hue.group" || lights.Dials[4].Control != "hue.brightness" {
		t.Fatal("lights page", lights.Keys[:12])
	}
}

func TestDefaultMediaPage(t *testing.T) {
	media := DefaultLayout().Pages[3]
	for n, id := range []string{"nowplaying.prev", "nowplaying.toggle", "nowplaying.next", "nowplaying.mute", resetFocusID, "", "appaudio.deck-apps", "nowplaying.deck-media"} {
		if media.Keys[27+n].Control != id {
			t.Fatalf("bottom row key %d is %q, want %q", 28+n, media.Keys[27+n].Control, id)
		}
	}
	if media.Dials[0].Control != "audio.gain-playback" || media.Dials[1].Control != "nowplaying.dial" || media.Dials[2].Control != "appaudio.focus" {
		t.Fatal("media page dials", media.Dials)
	}
	// The four filter states: a filtered category is Hidden, as its plugin
	// publishes it, and the other takes its rows and dials.
	sessions := func(n int, hidden bool) []snoofer.Control {
		var out []snoofer.Control
		for i := 0; i < n; i++ {
			out = append(out, snoofer.Control{ID: fmt.Sprintf("nowplaying.s-%02d", i), Label: "S", Collection: "nowplaying.sessions", Order: i + 1, Operations: []string{"press", "set"}, Hidden: hidden})
		}
		return out
	}
	apps := func(n int, hidden bool) []snoofer.Control {
		out := appControls(n)
		for i := range out {
			out[i].Hidden = hidden
		}
		return out
	}
	// The media dial and the focus dial hide with their filters; Playback stays.
	dial := func(media, apps bool) []snoofer.Control {
		return []snoofer.Control{{ID: "nowplaying.dial", Operations: []string{"adjust", "press"}, Hidden: !media},
			{ID: "appaudio.focus", Operations: []string{"adjust", "press"}, Hidden: !apps}, {ID: "audio.gain-playback", Operations: []string{"adjust"}}}
	}
	for _, c := range []struct {
		name               string
		controls           []snoofer.Control
		rows               []string
		firstAppDial, apps int
	}{
		{"both", append(append(sessions(3, false), apps(6, false)...), dial(true, true)...), []string{"nnn.....", "aaaaaa..", "........"}, 3, 2},
		{"apps off", append(append(sessions(20, false), apps(6, true)...), dial(true, false)...), []string{"nnnnnnnn", "nnnnnnnn", "nnnn...."}, -1, 0},
		{"media off", append(append(sessions(3, true), apps(20, false)...), dial(false, true)...), []string{"aaaaaaaa", "aaaaaaaa", "aaaa...."}, 1, 3},
		{"both off", append(append(sessions(3, true), apps(6, true)...), dial(false, false)...), []string{"........", "........", "........"}, -1, 0},
	} {
		page := DefaultLayout().expanded(c.controls).effective("media")
		var rows []string
		for row := 0; row < 3; row++ {
			line := ""
			for col := 0; col < 8; col++ {
				switch id := page.Keys[row*Columns+col].Control; {
				case strings.HasPrefix(id, "nowplaying.s-"):
					line += "n"
				case strings.HasPrefix(id, "appaudio.app-"):
					line += "a"
				default:
					line += "."
				}
			}
			rows = append(rows, line)
		}
		if strings.Join(rows, "|") != strings.Join(c.rows, "|") {
			t.Errorf("%s: rows %v, want %v", c.name, rows, c.rows)
		}
		count, first := 0, -1
		for n, b := range page.Dials {
			if strings.HasPrefix(b.Control, "appaudio.app-") {
				count++
				if first < 0 {
					first = n
				}
			}
		}
		if count != c.apps || first != c.firstAppDial {
			t.Errorf("%s: %d app dials from %d, want %d from %d", c.name, count, first, c.apps, c.firstAppDial)
		}
	}
}

func TestLayoutJSONRoundTrip(t *testing.T) {
	saved := snoofer.MarshalSettings(Settings{Layout: DefaultLayout()})
	var decoded Settings
	if err := snoofer.DecodeSettings(saved, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded.Layout, DefaultLayout()) {
		t.Fatal("layout changed through JSON", decoded.Layout.Pages[1].Regions)
	}
}
