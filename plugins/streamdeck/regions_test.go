package streamdeck

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"sound-snoofer/snoofer"
)

func clips(n int) []snoofer.Control {
	var out []snoofer.Control
	for i := 0; i < n; i++ {
		out = append(out, snoofer.Control{ID: fmt.Sprintf("soundboard.clip-%02d", i), Label: fmt.Sprintf("Clip %02d", i), Collection: "soundboard.clips", Operations: []string{"press"}})
	}
	return out
}

func TestRegionCells(t *testing.T) {
	// Corners in either order describe the same rectangle.
	if got := (Region{First: 20, Last: 0}).cells(); !reflect.DeepEqual(got, []int{0, 1, 2, 9, 10, 11, 18, 19, 20}) {
		t.Fatal(got)
	}
}

func TestRegionsKeepFixedFrameAndOverflow(t *testing.T) {
	// Clips fill r1–r4 c1–c8; column 9 holds the fixed frame.
	p := Page{ID: "sounds", Name: "Soundboard", Regions: []Region{{Source: "soundboard.clips", First: 0, Last: 34}}}
	p.Keys[8] = Binding{Control: "soundboard.overlap"}
	p.Keys[35] = Binding{Control: "soundboard.stop"}
	p.Keys[10] = Binding{Control: "soundboard.clip-05"} // A manual cell inside the region.
	l := Layout{Home: "sounds", Pages: []Page{p}}
	if err := l.Validate(nil); err != nil {
		t.Fatal(err)
	}
	expanded := l.expanded(clips(70))
	// 31 free cells; clip-05 is bound manually, leaving 69 candidates: 3 pages.
	if len(expanded.Pages) != 3 {
		t.Fatalf("pages %d", len(expanded.Pages))
	}
	for _, page := range expanded.Pages {
		if page.Keys[8].Control != "soundboard.overlap" || page.Keys[35].Control != "soundboard.stop" || page.Keys[10].Control != "soundboard.clip-05" {
			t.Fatalf("frame moved on %s", page.ID)
		}
		for _, key := range []int{17, 26} {
			if page.Keys[key].Control != "" {
				t.Fatalf("filled outside the region on %s key %d", page.ID, key)
			}
		}
	}
	first := expanded.Pages[0].Keys
	if first[0].Control != "soundboard.clip-00" || first[5].Control != "soundboard.clip-06" || first[9].Control != "soundboard.clip-09" || first[11].Control != "soundboard.clip-10" {
		t.Fatal("row-major fill skipping the manual cell", first[:12])
	}
	last := expanded.Pages[2]
	if last.ID != "sounds~auto~3" || last.Name != "Soundboard 3" || last.Keys[6].Control == "" || last.Keys[7].Control != "" {
		t.Fatal("last overflow page", last.Keys[:12])
	}
}

func TestRegionsSameSourceFillInSequence(t *testing.T) {
	p := Page{ID: "p", Name: "P", Regions: []Region{{Source: "soundboard.clips", First: 0, Last: 1}, {Source: "soundboard.clips", First: 27, Last: 28}}}
	expanded := Layout{Home: "p", Pages: []Page{p}}.expanded(clips(6))
	if len(expanded.Pages) != 2 {
		t.Fatal(len(expanded.Pages))
	}
	got := []string{expanded.Pages[0].Keys[0].Control, expanded.Pages[0].Keys[28].Control, expanded.Pages[1].Keys[0].Control}
	if !reflect.DeepEqual(got, []string{"soundboard.clip-00", "soundboard.clip-03", "soundboard.clip-04"}) {
		t.Fatal(got)
	}
}

func TestRegionsSkipHiddenAndParallelOverflow(t *testing.T) {
	scenes := []snoofer.Control{
		{ID: "hue.room-scene-1", Label: "Bright", Collection: "hue.room-scenes", Operations: []string{"press"}},
		{ID: "hue.room-scene-2", Collection: "hue.room-scenes", Operations: []string{"press"}, Hidden: true},
	}
	p := Page{ID: "mixed", Name: "Mixed", Regions: []Region{{Source: "hue.room-scenes", First: 0, Last: 1}, {Source: "soundboard.clips", First: 9, Last: 10}}}
	expanded := Layout{Home: "mixed", Pages: []Page{p}}.expanded(append(scenes, clips(3)...))
	if len(expanded.Pages) != 2 || expanded.Pages[0].Keys[1].Control != "" {
		t.Fatal("hidden scene placed or overflow wrong", len(expanded.Pages))
	}
	// The scenes region is exhausted on page 2; the clips region continues.
	if expanded.Pages[1].Keys[0].Control != "" || expanded.Pages[1].Keys[9].Control != "soundboard.clip-02" {
		t.Fatal(expanded.Pages[1].Keys[:11])
	}
}

func TestRegionValidation(t *testing.T) {
	for _, c := range []struct {
		page Page
		want string
	}{
		{Page{ID: "a", Name: "A", Regions: []Region{{Source: "x.y", First: 0, Last: 10}, {Source: "x.z", First: 10, Last: 20}}}, "overlap"},
		{Page{ID: "a", Name: "A", Regions: []Region{{Source: "x.y", First: 0, Last: 36}}}, "outside"},
		{Page{ID: "a", Name: "A", Regions: []Region{{Source: "nodot", First: 0, Last: 1}}}, "source"},
		{Page{ID: "a", Name: "A", AutoControls: "x.", Regions: []Region{{Source: "x.y", First: 0, Last: 1}}}, "both"},
	} {
		err := Layout{Home: "a", Pages: []Page{c.page}}.Validate(nil)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("want %q, got %v", c.want, err)
		}
	}
	full := Page{ID: "a", Name: "A", Regions: []Region{{Source: "x.y", First: 0, Last: 0}}}
	full.Keys[0] = Binding{Control: "manual"}
	if err := (Layout{Home: "a", Pages: []Page{full}}).Validate(nil); err == nil || !strings.Contains(err.Error(), "no free key") {
		t.Errorf("region without free key: %v", err)
	}
}

func TestLegacyPrefixMatchesWholePageRegion(t *testing.T) {
	legacy := Page{ID: "s", Name: "S", AutoControls: "soundboard.clip-"}
	legacy.Keys[35] = Binding{Control: "soundboard.stop"}
	region := legacy
	region.AutoControls = ""
	region.Regions = []Region{{Source: "soundboard.clip-", First: 0, Last: Keys - 1}}
	controls := clips(50)
	a := Layout{Home: "s", Pages: []Page{legacy}}.expanded(controls)
	b := Layout{Home: "s", Pages: []Page{region}}.expanded(controls)
	if len(a.Pages) != 2 || len(b.Pages) != 2 {
		t.Fatal("page counts", len(a.Pages), len(b.Pages))
	}
	for n := range a.Pages {
		if a.Pages[n].Keys != b.Pages[n].Keys {
			t.Fatal("legacy prefix and prefix region differ on page", n)
		}
	}
}

func TestEditorViewRegions(t *testing.T) {
	p := Page{ID: "s", Name: "S", Regions: []Region{{Source: "soundboard.clips", First: 0, Last: 1}}}
	l := Layout{Home: "s", Pages: []Page{p}}
	controls := clips(1)
	controls[0].CollectionLabel = "Soundboard clips"
	view := editorView(l, "s", 0, false, controls)
	if len(view.Regions) != 1 || view.Regions[0].Label != "Soundboard clips" || view.Regions[0].Legacy {
		t.Fatal(view.Regions)
	}
	if view.Keys[0].Region != 0 || view.Keys[1].Region != 0 || view.Keys[2].Region != -1 || view.Keys[0].Source != "Auto" {
		t.Fatal(view.Keys[:3])
	}
	legacy := editorView(Layout{Home: "s", Pages: []Page{{ID: "s", Name: "S", AutoControls: "soundboard.clip-"}}}, "s", 0, false, controls)
	if len(legacy.Regions) != 1 || !legacy.Regions[0].Legacy || legacy.Keys[35].Region != 0 {
		t.Fatal(legacy.Regions)
	}
}
