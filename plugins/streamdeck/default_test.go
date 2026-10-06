package streamdeck

import (
	"fmt"
	"reflect"
	"testing"

	"sound-snoofer/snoofer"
)

func TestDefaultLayoutPages(t *testing.T) {
	l := DefaultLayout()
	if err := l.Validate(nil); err != nil {
		t.Fatal(err)
	}
	if len(l.Pages) != 3 || l.Pages[0].ID != "home" || l.Pages[1].ID != "soundboard" || l.Pages[2].ID != "lights" {
		t.Fatal("default pages", l.Pages)
	}
	if l.Pages[0].Keys[35].Control != gotoPrefix+"lights" || l.Pages[0].Keys[34].Control != gotoPrefix+"soundboard" {
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
	if len(expanded.Pages) != 4 || expanded.Pages[2].ID != "soundboard~auto~2" {
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
