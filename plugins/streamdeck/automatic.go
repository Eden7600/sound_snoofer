package streamdeck

import (
	"fmt"
	"slices"
	"strings"

	"sound-snoofer/snoofer"
)

// stream is the cells and candidates of every region on a page that shares a
// source; such regions fill in sequence.
type stream struct {
	cells   []int
	matches []snoofer.Control
}

// expanded fills regions' free keys; saved, manual and shared bindings remain
// intact. Overflow adds pages that repeat the page's fixed bindings, each
// showing the next chunk of every region.
func (l Layout) expanded(controls []snoofer.Control) Layout {
	result := l.clone()
	result.Pages = nil
	for _, page := range l.Pages {
		fills := page.fills()
		if len(fills) == 0 {
			result.Pages = append(result.Pages, page)
			continue
		}
		bound := map[string]bool{}
		for n, b := range page.Keys {
			bound[b.Control] = true
			bound[l.SharedKeys[n].Control] = true
		}
		type sourceKey struct {
			source string
			prefix bool
		}
		var streams []*stream
		bySource := map[sourceKey]*stream{}
		for _, f := range fills {
			key := sourceKey{f.Source, f.prefix}
			s, ok := bySource[key]
			if !ok {
				s = &stream{matches: candidates(f, controls, bound)}
				bySource[key] = s
				streams = append(streams, s)
			}
			for _, cell := range f.cells() {
				if page.Keys[cell].Control == "" && l.SharedKeys[cell].Control == "" {
					s.cells = append(s.cells, cell)
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
					generated.Keys[cell] = Binding{Control: c.ID, Label: c.Label}
				}
			}
			result.Pages = append(result.Pages, generated)
		}
	}
	return result
}

// candidates lists a region's pressable, visible members that are not bound
// elsewhere on the page, in label order.
func candidates(f fill, controls []snoofer.Control, bound map[string]bool) []snoofer.Control {
	var out []snoofer.Control
	for _, c := range controls {
		if f.matches(c) && slices.Contains(c.Operations, "press") && !c.Hidden && !bound[c.ID] {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b snoofer.Control) int {
		if cmp := strings.Compare(a.Label, b.Label); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out
}
