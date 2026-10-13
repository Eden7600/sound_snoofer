package streamdeck

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

func TestShapesFollowModels(t *testing.T) {
	for model, want := range map[string]shape{
		"":        {columns: 9, keys: 36, dials: 5, pagesKey: -1},
		"plus-xl": {columns: 9, keys: 36, dials: 5, pagesKey: -1},
		"plus":    {columns: 4, keys: 8, dials: 3, pagesKey: -1},
		"xl":      {columns: 8, keys: 32, dials: 0, pagesKey: 31},
		"mk2":     {columns: 5, keys: 15, dials: 0, pagesKey: 14},
		"neo":     {columns: 4, keys: 8, dials: 0, pagesKey: -1},
	} {
		if got := shapeOf(model); got != want {
			t.Fatalf("%q: %+v, want %+v", model, got, want)
		}
	}
}

func TestEveryModelDefaultValidates(t *testing.T) {
	for _, m := range device.Models() {
		l := DefaultLayoutFor(m.ID)
		if l.model != modelKey(m.ID) {
			t.Fatalf("%s default is for %q", m.ID, l.model)
		}
		if err := l.Validate(nil); err != nil {
			t.Fatalf("%s: %v", m.ID, err)
		}
		for _, id := range []string{"home", "soundboard", "lights", "media", "meetings"} {
			if l.index(id) < 0 {
				t.Fatalf("%s lacks page %s", m.ID, id)
			}
		}
	}
}

func TestLayoutJSONKeepsTheGeometry(t *testing.T) {
	legacy, err := json.Marshal(Settings{Layout: DefaultLayout()})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeSettings(legacy)
	if err != nil || !reflect.DeepEqual(decoded.Layout.Pages, DefaultLayout().Pages) {
		t.Fatal("+ XL layout did not round-trip", err)
	}
	if !strings.Contains(string(legacy), `"dials":[{`) || strings.Count(string(legacy), `"control"`) < Keys {
		t.Fatal("+ XL positions not saved in full")
	}
	plus, err := json.Marshal(DefaultLayoutFor("plus"))
	if err != nil {
		t.Fatal(err)
	}
	var raw layoutJSON
	if err := json.Unmarshal(plus, &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw.Pages[0].Keys) != 8 || len(raw.Pages[0].Dials) != 3 || len(raw.SharedKeys) != 8 || len(raw.SharedDials) != 3 {
		t.Fatalf("+ layout saved %d keys, %d dials", len(raw.Pages[0].Keys), len(raw.Pages[0].Dials))
	}
}

func TestValidationRejectsPositionsOffTheDeck(t *testing.T) {
	bind := func(model string, edit func(*Layout)) error {
		l := Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}}}.forModel(model)
		edit(&l)
		return l.Validate(nil)
	}
	cases := map[string]struct {
		model string
		edit  func(*Layout)
		want  string
	}{
		"key past +":       {"plus", func(l *Layout) { l.Pages[0].Keys[8] = Binding{Control: "x.y"} }, "key 9 is not on this deck"},
		"dial past +":      {"plus", func(l *Layout) { l.Pages[0].Dials[3] = Binding{Control: "x.y"} }, "dial 4 is not on this deck"},
		"dial on MK.2":     {"mk2", func(l *Layout) { l.SharedDials[0] = Binding{Control: "x.y"} }, "dial 1 is not on this deck"},
		"Pages key":        {"mk2", func(l *Layout) { l.Pages[0].Keys[14] = Binding{Control: "x.y"} }, "key 15 is the Pages key"},
		"shared Pages key": {"xl", func(l *Layout) { l.SharedKeys[31] = Binding{Control: "x.y"} }, "key 32 is the Pages key"},
		"region past +":    {"plus", func(l *Layout) { l.Pages[0].Regions = []Region{{Source: "x.c", First: 0, Last: 9}} }, "outside the deck"},
		"dial region past": {"plus", func(l *Layout) { l.Pages[0].DialRegions = []Region{{Source: "x.c", First: 0, Last: 3}} }, "outside the deck"},
	}
	for name, c := range cases {
		if err := bind(c.model, c.edit); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// A region may span the Pages key, and on the + XL key 36 is ordinary.
	if err := bind("mk2", func(l *Layout) { l.Pages[0].Regions = []Region{{Source: "x.c", First: 10, Last: 14}} }); err != nil {
		t.Fatal(err)
	}
	if err := bind("", func(l *Layout) { l.Pages[0].Keys[35] = Binding{Control: "x.y"} }); err != nil {
		t.Fatal(err)
	}
}

func TestRegionsNeverFillThePagesKey(t *testing.T) {
	var clips []snoofer.Control
	for n := 0; n < 30; n++ {
		clips = append(clips, snoofer.Control{ID: fmt.Sprintf("soundboard.clip-%02d", n), Label: "Clip", Collection: "soundboard.clips",
			Kind: "command", Operations: []string{"press"}, Available: true})
	}
	l := Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A", Regions: []Region{{Source: "soundboard.clips", First: 0, Last: 14}}}}}.forModel("mk2")
	expanded := l.expanded(clips)
	if len(expanded.Pages) != 3 {
		t.Fatalf("%d sets for 30 clips on 14 keys", len(expanded.Pages))
	}
	for _, p := range expanded.Pages {
		if p.Keys[14].Control != "" {
			t.Fatalf("%s filled the Pages key", p.ID)
		}
		if p.Keys[15].Control != "" {
			t.Fatalf("%s filled past the deck", p.ID)
		}
	}
}

func TestSettingsAssignModels(t *testing.T) {
	own := DefaultLayoutFor("plus")
	raw := snoofer.MarshalSettings(Settings{Layout: DefaultLayout(),
		Models:  map[string]Layout{"mk2": DefaultLayoutFor("mk2")},
		Serials: map[string]Layout{"PLUS1": own, "LEGACY": DefaultLayout()},
		Devices: map[string]Deck{"PLUS1": {Model: "plus", Name: "Desk"}}})
	s, err := decodeSettings(raw)
	if err != nil {
		t.Fatal(err)
	}
	if s.Models["mk2"].model != "mk2" || s.Serials["PLUS1"].model != "plus" || s.Serials["LEGACY"].model != "" {
		t.Fatal("models not assigned")
	}
	if err := s.validate(nil); err != nil {
		t.Fatal(err)
	}
	if got := s.layoutFor("plus", "PLUS1"); got.model != "plus" || len(got.Pages) != len(own.Pages) {
		t.Fatal("own layout not used")
	}
	// A + XL override never lands on a deck of another model.
	if got := s.layoutFor("plus", "LEGACY"); got.model != "plus" {
		t.Fatal("+ XL override used on a +")
	}
	if got := s.layoutFor("neo", ""); got.model != "neo" {
		t.Fatal("neo default not used")
	}
	if got := s.layoutFor("mk2", "OTHER"); !reflect.DeepEqual(got.Pages, s.Models["mk2"].Pages) {
		t.Fatal("saved model layout not used")
	}

	for name, bad := range map[string]Settings{
		"unknown model":  {Layout: DefaultLayout(), Models: map[string]Layout{"pedal": DefaultLayout()}},
		"+ XL in models": {Layout: DefaultLayout(), Models: map[string]Layout{"plus-xl": DefaultLayout()}},
		"deck model":     {Layout: DefaultLayout(), Devices: map[string]Deck{"A": {Model: "mini"}}},
		"paging":         {Layout: DefaultLayout(), Paging: "sometimes"},
	} {
		if _, err := decodeSettings(snoofer.MarshalSettings(bad)); err == nil {
			t.Fatal(name, "accepted")
		}
	}
	// A deck's layout using positions its model lacks fails validation.
	s.Serials["PLUS1"] = DefaultLayout().forModel("plus")
	if err := s.validate(nil); err == nil {
		t.Fatal("+ XL layout accepted as a +")
	}
}
