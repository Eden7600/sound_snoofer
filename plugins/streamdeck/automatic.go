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

// stack is a region whose rows several sources share by need (Region.Sources).
type stack struct {
	rows    [][]int             // Free cells by row, top to bottom.
	sources [][]snoofer.Control // Candidates of each source, in order.
}

// sets places a stack's sources set by set. In each set every source with
// members left gets one row while rows last, then leftover rows go to sources
// in order up to what they need; each source fills its rows in reading order.
func (s stack) sets() []map[int]snoofer.Control {
	if len(s.rows) == 0 {
		return nil
	}
	width := 0
	for _, row := range s.rows {
		width = max(width, len(row))
	}
	remaining := make([][]snoofer.Control, len(s.sources))
	copy(remaining, s.sources)
	var out []map[int]snoofer.Control
	for {
		var active []int
		for n, members := range remaining {
			if len(members) > 0 {
				active = append(active, n)
			}
		}
		if len(active) == 0 {
			return out
		}
		rows := make([]int, len(s.sources))
		left := len(s.rows)
		for _, n := range active {
			if left > 0 {
				rows[n]++
				left--
			}
		}
		for _, n := range active {
			need := (len(remaining[n]) + width - 1) / width
			extra := min(left, need-rows[n])
			if extra > 0 {
				rows[n] += extra
				left -= extra
			}
		}
		placed := map[int]snoofer.Control{}
		row := 0
		for n := range s.sources {
			for r := 0; r < rows[n]; r++ {
				for _, cell := range s.rows[row] {
					if len(remaining[n]) == 0 {
						break
					}
					placed[cell] = remaining[n][0]
					remaining[n] = remaining[n][1:]
				}
				row++
			}
		}
		out = append(out, placed)
	}
}

// expanded fills regions' free keys and dials; saved, manual and shared
// bindings remain intact unless their control is Hidden, which frees the
// position for a covering region. Overflow adds pages that repeat the page's
// fixed bindings, each showing the next chunk of every region, so key and
// dial regions of one collection page in step.
func (l Layout) expanded(controls []snoofer.Control) Layout {
	result := l.clone()
	result.Pages = nil
	hidden := map[string]bool{}
	mirrors := map[string]string{} // Stand-in control ID to the control it mirrors.
	for _, c := range controls {
		if c.Mirrors != "" && !c.Hidden {
			mirrors[c.ID] = c.Mirrors
		}
		// Stream Deck's own keys never yield: their visibility depends on
		// this expansion.
		if c.Hidden && !strings.HasPrefix(c.ID, "streamdeck.") {
			hidden[c.ID] = true
		}
	}
	free := func(bindings ...Binding) bool {
		for _, b := range bindings {
			if b.Control != "" && !hidden[b.Control] {
				return false
			}
		}
		return true
	}
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
			// A dial standing in for a control keeps that control off the
			// page's other dials.
			boundDials[mirrors[b.Control]] = true
			boundDials[mirrors[l.SharedDials[n].Control]] = true
		}
		delete(boundDials, "")
		type sourceKey struct {
			source        string
			prefix, dials bool
		}
		var streams []*stream
		var stacks []stack
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
			if len(f.Sources) > 0 {
				st := stack{}
				for _, source := range append([]string{f.Source}, f.Sources...) {
					st.sources = append(st.sources, candidates(fill{Region: Region{Source: source}}, controls, boundKeys, "press"))
				}
				for _, row := range f.rows() {
					var cells []int
					for _, cell := range row {
						if free(page.Keys[cell], l.SharedKeys[cell]) {
							cells = append(cells, cell)
						}
					}
					if len(cells) > 0 {
						st.rows = append(st.rows, cells)
					}
				}
				stacks = append(stacks, st)
				continue
			}
			s := add(f, false)
			for _, cell := range f.cells() {
				if free(page.Keys[cell], l.SharedKeys[cell]) {
					s.cells = append(s.cells, cell)
				}
			}
		}
		for _, r := range page.DialRegions {
			s := add(fill{Region: r}, true)
			for _, dial := range r.dialCells() {
				if free(page.Dials[dial], l.SharedDials[dial]) {
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
		stacked := make([][]map[int]snoofer.Control, len(stacks))
		for n, st := range stacks {
			stacked[n] = st.sets()
			pages = max(pages, len(stacked[n]))
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
			for _, sets := range stacked {
				if number > len(sets) {
					continue
				}
				for cell, c := range sets[number-1] {
					generated.Keys[cell] = Binding{Control: c.ID, Label: c.Label}
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
