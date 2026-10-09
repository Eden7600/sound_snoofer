package streamdeck

import (
	"fmt"
	"testing"

	"sound-snoofer/snoofer"
)

func TestClipEditToggleOverflow(t *testing.T) {
	p := Page{ID: "p", Name: "P", Regions: []Region{{Source: "a.s", First: 0, Last: 1}}, DialRegions: []Region{{Source: "a.s", First: 0, Last: 1}}}
	l := Layout{Home: "p", Pages: []Page{p}}
	controls := members("a.s", 6)
	if sets := len(l.expanded(controls).sets("p")); sets != 3 {
		t.Fatal("unclipped sets", sets)
	}
	clipped, err := l.editRegion(0, "clip", "0,on")
	if err == nil {
		clipped, err = clipped.editRegion(0, "clip", fmt.Sprintf("%d,on", 1))
	}
	if err != nil || !clipped.Pages[0].Regions[0].Clip || !clipped.Pages[0].DialRegions[0].Clip {
		t.Fatal("clip on", clipped.Pages[0], err)
	}
	if l.Pages[0].Regions[0].Clip || l.Pages[0].DialRegions[0].Clip {
		t.Fatal("clip edit leaked into the original draft")
	}
	if err := clipped.Validate(nil); err != nil {
		t.Fatal(err)
	}
	if sets := len(clipped.expanded(controls).sets("p")); sets != 1 {
		t.Fatal("clipped sets", sets)
	}
	restored, err := clipped.editRegion(0, "clip", "0, off")
	if err != nil || restored.Pages[0].Regions[0].Clip {
		t.Fatal("clip off", restored.Pages[0].Regions, err)
	}
	for _, bad := range []string{"2,on", "0", "0,yes", "x,on"} {
		if _, err := l.editRegion(0, "clip", bad); err == nil {
			t.Errorf("clip %q accepted", bad)
		}
	}
}

func TestClippedStackShowsFirstSet(t *testing.T) {
	l := stackedPage()
	controls := append(members("a.s", 30), members("b.s", 3)...)
	if sets := len(l.expanded(controls).sets("p")); sets != 2 {
		t.Fatal("unclipped stack sets", sets)
	}
	l.Pages[0].Regions[0].Clip = true
	expanded := l.expanded(controls)
	if sets := len(expanded.sets("p")); sets != 1 {
		t.Fatal("clipped stack sets", sets)
	}
	if rows := rowsOf(expanded.effective("p")); rows[0] != "aaaaaaaa" || rows[2] != "bbb....." {
		t.Fatal("clipped stack first set", rows)
	}
}

func TestEditorViewSetsAndRegions(t *testing.T) {
	l := stackedPage()
	l.Pages[0].DialRegions = []Region{{Source: "b.s", First: 0, Last: 1, Clip: true}}
	controls := append(members("a.s", 30), members("b.s", 3)...)
	controls = append(controls, snoofer.Control{ID: "b.s-label", Collection: "b.s", CollectionLabel: "Apps", Hidden: true})
	view := editorView(l, "p", 0, false, controls)
	if view.Sets != 2 || view.Scrolls {
		t.Fatal("sets", view.Sets, view.Scrolls)
	}
	if view.Regions[0].Stacked != "Apps" || view.Regions[0].Clip || !view.Regions[1].Clip || view.Regions[1].Stacked != "" {
		t.Fatal("regions", view.Regions)
	}
	l.Pages[0].Keys[35] = Binding{Control: scrollPrefix + "down"}
	if view := editorView(l, "p", 0, false, controls); !view.Scrolls {
		t.Fatal("scroll key not reported")
	}
	// Without the source's members the count follows the remaining controls.
	if view := editorView(l, "p", 0, false, members("b.s", 3)); view.Sets != 1 {
		t.Fatal("sets without a source", view.Sets)
	}
}
