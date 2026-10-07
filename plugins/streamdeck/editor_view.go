package streamdeck

import (
	"slices"
	"strings"

	"sound-snoofer/snoofer"
)

// EditorView is read-only presentation of the plugin-owned draft.
type EditorView struct {
	Keys        []EditorSlot
	Dials       []EditorSlot
	Regions     []EditorRegion
	Collections []EditorCollection // Sources the editor offers for regions.
	Selected    int
	Dirty       bool
	Home        bool
}
type EditorSlot struct {
	Control, Label, Source string
	Region                 int // Index into Regions, or -1.
}

// EditorCollection is a region source published by some control.
type EditorCollection struct {
	ID, Label string
}

// EditorRegion is a page region with its editor label. Legacy marks the
// whole-page region implied by an automatic prefix; Dials marks a dial
// region, whose First and Last are dial indexes.
type EditorRegion struct {
	Source, Label string
	First, Last   int
	Legacy, Dials bool
}

func editorView(draft Layout, page string, selected int, dirty bool, controls []snoofer.Control) EditorView {
	view := EditorView{Selected: selected, Dirty: dirty, Home: draft.Home == page}
	base := draft.Pages[draft.index(page)]
	effective := draft.expanded(controls).effective(page)
	region := make([]int, Keys)
	for n := range region {
		region[n] = -1
	}
	for n, f := range base.fills() {
		view.Regions = append(view.Regions, EditorRegion{Source: f.Source, Label: collectionLabel(f, controls), First: f.First, Last: f.Last, Legacy: len(base.Regions) == 0})
		for _, cell := range f.cells() {
			region[cell] = n
		}
	}
	seen := map[string]bool{}
	for _, c := range controls {
		if c.Collection != "" && !seen[c.Collection] {
			seen[c.Collection] = true
			label := c.CollectionLabel
			if label == "" {
				label = c.Collection
			}
			view.Collections = append(view.Collections, EditorCollection{ID: c.Collection, Label: label})
		}
	}
	slices.SortFunc(view.Collections, func(a, b EditorCollection) int { return strings.Compare(a.Label, b.Label) })
	slot := func(binding, manual, shared Binding, region int) EditorSlot {
		source := ""
		if shared.Control != "" {
			source = "Shared"
		} else if binding.Control != "" && manual.Control == "" {
			source = "Auto"
		}
		return EditorSlot{Control: binding.Control, Label: binding.Label, Source: source, Region: region}
	}
	for n, b := range effective.Keys {
		view.Keys = append(view.Keys, slot(b, base.Keys[n], draft.SharedKeys[n], region[n]))
	}
	dialRegion := make([]int, Dials)
	for n := range dialRegion {
		dialRegion[n] = -1
	}
	for _, r := range base.DialRegions {
		for _, dial := range r.dialCells() {
			dialRegion[dial] = len(view.Regions)
		}
		view.Regions = append(view.Regions, EditorRegion{Source: r.Source, Label: collectionLabel(fill{Region: r}, controls), First: r.First, Last: r.Last, Dials: true})
	}
	for n, b := range effective.Dials {
		view.Dials = append(view.Dials, slot(b, base.Dials[n], draft.SharedDials[n], dialRegion[n]))
	}
	return view
}

// collectionLabel names a region's source: the collection's published label,
// or the raw prefix for prefix regions and collections with no members.
func collectionLabel(f fill, controls []snoofer.Control) string {
	if f.prefix {
		return f.Source
	}
	labels := []string{}
	for _, source := range append([]string{f.Source}, f.Sources...) {
		label := source
		for _, c := range controls {
			if c.Collection == source && c.CollectionLabel != "" {
				label = c.CollectionLabel
				break
			}
		}
		labels = append(labels, label)
	}
	return strings.Join(labels, " + ")
}
