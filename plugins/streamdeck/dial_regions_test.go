package streamdeck

import (
	"fmt"
	"strings"
	"testing"

	"sound-snoofer/snoofer"
)

func appControls(n int) []snoofer.Control {
	var out []snoofer.Control
	for i := 1; i <= n; i++ {
		out = append(out, snoofer.Control{ID: fmt.Sprintf("appaudio.app-%d", i), Label: fmt.Sprintf("App %d", 10-i), Collection: "appaudio.apps", Order: i,
			Operations: []string{"press", "adjust", "set"}})
	}
	return out
}

func TestDialRegionsPageWithKeys(t *testing.T) {
	l := DefaultLayout()
	if err := l.Validate(nil); err != nil {
		t.Fatal(err)
	}
	// Six apps on four free dials (the media dial is shown): two dial sets.
	expanded := l.expanded(append(appControls(6), snoofer.Control{ID: "nowplaying.dial", Operations: []string{"adjust"}}))
	first, second := expanded.effective("media"), expanded.effective("media~auto~2")
	if second.ID != "media~auto~2" {
		t.Fatal("six apps need a second dial set")
	}
	for n := 0; n < 4; n++ {
		if first.Dials[1+n].Control != fmt.Sprintf("appaudio.app-%d", n+1) {
			t.Fatalf("set 1 dial %d: %s", n+2, first.Dials[1+n].Control)
		}
	}
	if second.Dials[1].Control != "appaudio.app-5" || second.Dials[3].Control != "" || second.Dials[0].Control != "nowplaying.dial" || second.Keys[17].Control != scrollPrefix+"up" {
		t.Fatal("set 2", second.Dials)
	}
	// Dial regions take only adjustable members.
	pressOnly := []snoofer.Control{{ID: "appaudio.app-x", Label: "X", Collection: "appaudio.apps", Operations: []string{"press"}}}
	if expanded := l.expanded(pressOnly).effective("media"); expanded.Keys[0].Control == "" || expanded.Dials[1].Control != "" {
		t.Fatal("press-only member placed on a dial")
	}
}

func TestDialRegionValidationAndEdits(t *testing.T) {
	base := Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}}}
	for _, c := range []struct {
		regions []Region
		bound   bool
		want    string
	}{
		{[]Region{{Source: "appaudio.apps", First: 0, Last: 5}}, false, "outside"},
		{[]Region{{Source: "appaudio.apps", First: 0, Last: 2}, {Source: "x.y", First: 2, Last: 3}}, false, "overlap"},
		{[]Region{{Source: "appaudio.app-", First: 0, Last: 1}}, false, "collection source"},
		{[]Region{{Source: "appaudio.apps", First: 0, Last: 0}}, true, "no free dial"},
	} {
		l := base.clone()
		l.Pages[0].DialRegions = c.regions
		if c.bound {
			l.Pages[0].Dials[0] = Binding{Control: "audio.gain-playback"}
		}
		if err := l.Validate(nil); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.want, err)
		}
	}
	// Editor slots: dials follow the 36 keys; indexes span key then dial regions.
	l, err := base.editRegion(0, "add", "0,4,soundboard.clips")
	if err == nil {
		l, err = l.editRegion(0, "add", fmt.Sprintf("%d,%d,appaudio.apps", Keys, Keys+4))
	}
	if err != nil || len(l.Pages[0].DialRegions) != 1 || l.Pages[0].DialRegions[0].Last != 4 {
		t.Fatal("add dial region", err, l.Pages[0].DialRegions)
	}
	if _, err := l.editRegion(0, "add", fmt.Sprintf("30,%d,appaudio.apps", Keys)); err == nil {
		t.Fatal("mixed region accepted")
	}
	view := editorView(l, "a", 0, false, appControls(2))
	if len(view.Regions) != 2 || !view.Regions[1].Dials || view.Dials[3].Region != 1 || view.Keys[2].Region != 0 {
		t.Fatalf("editor view %+v %+v", view.Regions, view.Dials)
	}
	l, err = l.editRegion(0, "source", "1,hue.room-scenes")
	if err != nil || l.Pages[0].DialRegions[0].Source != "hue.room-scenes" {
		t.Fatal("source", err)
	}
	l, err = l.editRegion(0, "remove", "1")
	if err != nil || len(l.Pages[0].DialRegions) != 0 || len(l.Pages[0].Regions) != 1 {
		t.Fatal("remove", err)
	}
}
