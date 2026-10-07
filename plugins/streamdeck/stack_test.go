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
