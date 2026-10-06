package streamdeck

import (
	"fmt"
	"slices"
	"testing"

	"sound-snoofer/snoofer"
)

func TestScrollSets(t *testing.T) {
	page := Page{ID: "sounds", Name: "Soundboard", Regions: []Region{{Source: "soundboard.clips", First: 0, Last: 8}}}
	page.Keys[0] = Binding{Control: scrollPrefix + "up"}
	page.Keys[8] = Binding{Control: scrollPrefix + "down"}
	l := Layout{Home: "home", Pages: []Page{{ID: "home", Name: "Home"}, page, {ID: "lights", Name: "Lights"}}}
	if err := l.Validate(nil); err != nil {
		t.Fatal(err)
	}
	var clips []snoofer.Control
	for n := 0; n < 15; n++ {
		clips = append(clips, snoofer.Control{ID: fmt.Sprintf("soundboard.clip-%02d", n), Label: fmt.Sprintf("Clip %02d", n), Collection: "soundboard.clips", Operations: []string{"press"}})
	}
	expanded := l.expanded(clips)
	if sets := expanded.sets("sounds~auto~2"); !slices.Equal(sets, []string{"sounds", "sounds~auto~2", "sounds~auto~3"}) {
		t.Fatal("sets", sets)
	}
	if expanded.scroll("sounds", 1) != "sounds~auto~2" || expanded.scroll("sounds~auto~3", 1) != "sounds" || expanded.scroll("sounds", -1) != "sounds~auto~3" {
		t.Fatal("scroll does not step and wrap")
	}
	up := scrollControls(expanded, "sounds~auto~2")
	if up[0].Value != "2/3" || up[1].Value != "2/3" || up[0].Hidden || up[0].Icon != "deck-up" || up[1].ShortLabel != "Down" {
		t.Fatalf("scroll controls %+v", up)
	}
	if expanded.next("sounds~auto~3", 1) != "lights" || expanded.pageNames("sounds~auto~2") != [3]string{"Home", "Soundboard 2", "Lights"} {
		t.Fatal("page dial stops", expanded.pageNames("sounds~auto~2"))
	}
	single := l.expanded(clips[:7])
	if c := scrollControls(single, "sounds"); !c[0].Hidden || !c[1].Hidden || c[0].Value != "1/1" {
		t.Fatalf("single set %+v", c)
	}
	if scrollControls(single, "home")[0].Hidden != true {
		t.Fatal("page without overflow shows scroll keys")
	}
}
