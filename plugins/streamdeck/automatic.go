package streamdeck

import (
	"fmt"
	"slices"
	"strings"

	"sound-snoofer/snoofer"
)

// stream is the cells and candidates of every region on a page that shares a
// source and position kind; such regions fill in sequence.
type stream struct {
	dials   bool
	cells   []int
	matches []snoofer.Control
}

// expanded fills regions' free keys and dials; saved, manual and shared
// bindings remain intact. Overflow adds pages that repeat the page's fixed
// bindings, each showing the next chunk of every region, so key and dial
// regions of one collection page in step.
func (l Layout) expanded(controls []snoofer.Control) Layout {
	result := l.clone()
	result.Pages = nil
	for _, page := range l.Pages {
		fills := page.fills()
		if len(fills) == 0 && len(page.DialRegions) == 0 {
			result.Pages = append(result.Pages, page)
			continue
		}
		boundKeys, boundDials := map[string]bool{}, map[string]bool{}
		for n, b := range page.Keys {
			boundKeys[b.Control] = true
			boundKeys[l.SharedKeys[n].Control] = true
		}
		for n, b := range page.Dials {
			boundDials[b.Control] = true
			boundDials[l.SharedDials[n].Control] = true
		}
		type sourceKey struct {
			source        string
			prefix, dials bool
		}
		var streams []*stream
		bySource := map[sourceKey]*stream{}
		add := func(f fill, dials bool) *stream {
			key := sourceKey{f.Source, f.prefix, dials}
			s, ok := bySource[key]
			if !ok {
				if dials {
					s = &stream{dials: true, matches: candidates(f, controls, boundDials, "adjust")}
				} else {
					s = &stream{matches: candidates(f, controls, boundKeys, "press")}
				}
				bySource[key] = s
				streams = append(streams, s)
			}
			return s
		}
		for _, f := range fills {
			s := add(f, false)
			for _, cell := range f.cells() {
				if page.Keys[cell].Control == "" && l.SharedKeys[cell].Control == "" {
					s.cells = append(s.cells, cell)
				}
			}
		}
		for _, r := range page.DialRegions {
			s := add(fill{Region: r}, true)
			for _, dial := range r.dialCells() {
				if page.Dials[dial].Control == "" && l.SharedDials[dial].Control == "" {
					s.cells = append(s.cells, dial)
				}
			}
		}
		pages := 1
		for _, s := range streams {
			if len(s.cells) > 0 {
				pages = max(pages, (len(s.matches)+len(s.cells)-1)/len(s.cells))
			}
		}
		for number := 1; number <= pages; number++ {
			generated := page
			if number > 1 {
				generated.ID = fmt.Sprintf("%s~auto~%d", page.ID, number)
				generated.Name = fmt.Sprintf("%s %d", page.Name, number)
			}
			for _, s := range streams {
				offset := (number - 1) * len(s.cells)
				for n, cell := range s.cells {
					if offset+n >= len(s.matches) {
						break
					}
					c := s.matches[offset+n]
					if s.dials {
						generated.Dials[cell] = Binding{Control: c.ID, Label: c.Label}
					} else {
						generated.Keys[cell] = Binding{Control: c.ID, Label: c.Label}
					}
				}
			}
			result.Pages = append(result.Pages, generated)
		}
	}
	return result
}

// candidates lists a region's visible members that support op and are not
// bound elsewhere on the page, in collection order, then label order.
func candidates(f fill, controls []snoofer.Control, bound map[string]bool, op string) []snoofer.Control {
	var out []snoofer.Control
	for _, c := range controls {
		if f.matches(c) && slices.Contains(c.Operations, op) && !c.Hidden && !bound[c.ID] {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b snoofer.Control) int {
		if a.Order != b.Order {
			return a.Order - b.Order
		}
		if cmp := strings.Compare(a.Label, b.Label); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out
}
