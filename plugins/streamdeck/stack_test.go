package streamdeck

import (
	"fmt"
	"strings"
	"testing"

	"sound-snoofer/snoofer"
)

func members(collection string, n int) []snoofer.Control {
	var out []snoofer.Control
	for i := 1; i <= n; i++ {
		out = append(out, snoofer.Control{ID: fmt.Sprintf("%s-%02d", collection, i), Label: fmt.Sprintf("%s %02d", collection, i), Collection: collection,
			Order: i, Operations: []string{"press", "adjust"}})
	}
	return out
}

// rowsOf reports which collection fills each row of keys 0–26 (c1–c8).
func rowsOf(p Page) []string {
	var out []string
	for row := 0; row < 3; row++ {
		var parts []string
		for col := 0; col < 8; col++ {
			id := p.Keys[row*Columns+col].Control
			if id == "" {
				parts = append(parts, ".")
				continue
			}
			parts = append(parts, id[:1])
		}
		out = append(out, strings.Join(parts, ""))
	}
	return out
}

func stackedPage() Layout {
	p := Page{ID: "p", Name: "P", Regions: []Region{{Source: "a.s", Sources: []string{"b.s"}, First: 0, Last: 25}}}
	return Layout{Home: "p", Pages: []Page{p}}
}

func TestStackedRegionSharesRowsByNeed(t *testing.T) {
	l := stackedPage()
	if err := l.Validate(nil); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		a, b int
		want []string
	}{
		{"both small", 3, 2, []string{"aaa.....", "bb......", "........"}},
		{"a needs more", 12, 2, []string{"aaaaaaaa", "aaaa....", "bb......"}},
		{"b needs more", 2, 20, []string{"aa......", "bbbbbbbb", "bbbbbbbb"}},
		{"only a", 20, 0, []string{"aaaaaaaa", "aaaaaaaa", "aaaa...."}},
		{"only b", 0, 9, []string{"bbbbbbbb", "b.......", "........"}},
	} {
		page := l.expanded(append(members("a.s", c.a), members("b.s", c.b)...)).effective("p")
		if got := rowsOf(page); strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	// Overflow continues each source where it stopped.
	expanded := l.expanded(append(members("a.s", 30), members("b.s", 3)...))
	if len(expanded.Pages) != 2 {
		t.Fatal("sets", len(expanded.Pages))
	}
	second := expanded.effective("p~auto~2")
	if second.Keys[0].Control != "a.s-17" || rowsOf(second)[0] != "aaaaaaaa" || rowsOf(second)[1] != "aaaaaa.." {
		t.Fatal("second set", rowsOf(second), second.Keys[0])
	}
}

func TestHiddenBindingsYield(t *testing.T) {
	p := Page{ID: "p", Name: "P", Regions: []Region{{Source: "a.s", First: 0, Last: 2}}, DialRegions: []Region{{Source: "a.s", First: 0, Last: 1}}}
	p.Keys[1] = Binding{Control: "x.hidden"}
	p.Keys[2] = Binding{Control: "streamdeck.scroll-up"}
	p.Dials[0] = Binding{Control: "x.dial"}
	l := Layout{Home: "p", Pages: []Page{p}}
	controls := append(members("a.s", 5), snoofer.Control{ID: "x.hidden", Hidden: true}, snoofer.Control{ID: "streamdeck.scroll-up", Hidden: true},
		snoofer.Control{ID: "x.dial", Operations: []string{"adjust"}})
	page := l.expanded(controls).effective("p")
	if page.Keys[0].Control != "a.s-01" || page.Keys[1].Control != "a.s-02" {
		t.Fatal("hidden binding did not yield", page.Keys[:3])
	}
	if page.Keys[2].Control != "streamdeck.scroll-up" {
		t.Fatal("a Stream Deck key yielded", page.Keys[2])
	}
	if page.Dials[0].Control != "x.dial" || page.Dials[1].Control != "a.s-01" {
		t.Fatal("visible dial binding yielded", page.Dials[:2])
	}
	controls[len(controls)-1].Hidden = true
	if page := l.expanded(controls).effective("p"); page.Dials[0].Control != "a.s-01" {
		t.Fatal("hidden dial binding did not yield", page.Dials[:2])
	}
}

func TestStackedRegionValidationAndLabel(t *testing.T) {
	l := stackedPage()
	l.Pages[0].Regions[0].Sources = []string{"a.s"}
	if err := l.Validate(nil); err == nil || !strings.Contains(err.Error(), "invalid source") {
		t.Fatal("repeated source accepted", err)
	}
	l = stackedPage()
	controls := []snoofer.Control{{ID: "a.s-1", Collection: "a.s", CollectionLabel: "Media sessions"}, {ID: "b.s-1", Collection: "b.s", CollectionLabel: "Apps"}}
	if view := editorView(l, "p", 0, false, controls); view.Regions[0].Label != "Media sessions + Apps" {
		t.Fatal("label", view.Regions[0].Label)
	}
	// Draft edits do not share Sources with the saved layout.
	draft := l.clone()
	draft.Pages[0].Regions[0].Sources[0] = "c.s"
	if l.Pages[0].Regions[0].Sources[0] != "b.s" {
		t.Fatal("clone shares Sources")
	}
}

func TestMirroredControlNotRepeatedOnDials(t *testing.T) {
	p := Page{ID: "p", Name: "P", DialRegions: []Region{{Source: "a.s", First: 0, Last: 4}}}
	p.Dials[0] = Binding{Control: "x.focus"}
	l := Layout{Home: "p", Pages: []Page{p}}
	focus := snoofer.Control{ID: "x.focus", Mirrors: "a.s-01", Operations: []string{"adjust"}}
	page := l.expanded(append(members("a.s", 5), focus)).effective("p")
	for n, b := range page.Dials[1:] {
		if b.Control == "a.s-01" {
			t.Fatal("mirrored app repeated on dial", n+2)
		}
	}
	if page.Dials[1].Control != "a.s-02" {
		t.Fatal("next app does not take the dial", page.Dials)
	}
	// A hidden stand-in yields its dial and mirrors nothing.
	focus.Hidden = true
	if page := l.expanded(append(members("a.s", 5), focus)).effective("p"); page.Dials[0].Control != "a.s-01" {
		t.Fatal("hidden stand-in still excluded the app", page.Dials)
	}
}

func TestClippedRegionsNeverOverflow(t *testing.T) {
	home := DefaultLayout().Pages[0]
	l := Layout{Home: "home", Pages: []Page{home}}
	expanded := l.expanded(append(members("nowplaying.sessions", 6), members("appaudio.apps", 9)...))
	if len(expanded.Pages) != 1 {
		t.Fatal("clipped Home strips added overflow sets", len(expanded.Pages))
	}
	page := expanded.Pages[0]
	if page.Keys[18].Control != "nowplaying.sessions-01" || page.Keys[21].Control != "nowplaying.sessions-04" || page.Keys[22].Control != "appaudio.apps-01" || page.Keys[25].Control != "appaudio.apps-04" {
		t.Fatal("strips", page.Keys[18:26])
	}
	for n, id := range map[int]string{27: "nowplaying.prev", 28: "nowplaying.toggle", 29: "nowplaying.next", 30: "hue.brightness", 31: "hue.motion"} {
		if page.Keys[n].Control != id {
			t.Errorf("key %d is %q, want %q", n+1, page.Keys[n].Control, id)
		}
	}
	for _, b := range page.Keys {
		if b.Control == "core.open-controls" || strings.HasPrefix(b.Control, "hue.sync") {
			t.Error("removed control still on Home", b.Control)
		}
	}
	// A clipped region beside an unclipped one of the same source does not
	// stop the unclipped one from paging.
	p := Page{ID: "p", Name: "P", Regions: []Region{{Source: "a.s", First: 0, Last: 1, Clip: true}, {Source: "a.s", First: 9, Last: 10}}}
	if got := (Layout{Home: "p", Pages: []Page{p}}).expanded(members("a.s", 9)); len(got.Pages) != 3 {
		t.Fatal("mixed regions", len(got.Pages))
	}
}
