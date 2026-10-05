package streamdeck

import (
	"fmt"
	"slices"
	"strings"

	"sound-snoofer/snoofer"
)

// expanded fills only opted-in empty keys; saved/manual bindings remain intact.
func (l Layout) expanded(controls []snoofer.Control) Layout {
	result := l.clone()
	result.Pages = nil
	for _, page := range l.Pages {
		if page.AutoControls == "" {
			result.Pages = append(result.Pages, page)
			continue
		}
		free := []int{}
		bound := map[string]bool{}
		for n, b := range page.Keys {
			if b.Control == "" && l.SharedKeys[n].Control == "" {
				free = append(free, n)
			}
			bound[b.Control] = true
			bound[l.SharedKeys[n].Control] = true
		}
		if len(free) == 0 {
			result.Pages = append(result.Pages, page)
			continue
		}
		matches := []snoofer.Control{}
		for _, c := range controls {
			if strings.HasPrefix(c.ID, page.AutoControls) && slices.Contains(c.Operations, "press") && !bound[c.ID] {
				matches = append(matches, c)
			}
		}
		slices.SortFunc(matches, func(a, b snoofer.Control) int {
			if cmp := strings.Compare(a.Label, b.Label); cmp != 0 {
				return cmp
			}
			return strings.Compare(a.ID, b.ID)
		})
		for offset := 0; ; offset += len(free) {
			generated := page
			if offset > 0 {
				number := offset/len(free) + 1
				generated.ID = fmt.Sprintf("%s~auto~%d", page.ID, number)
				generated.Name = fmt.Sprintf("%s %d", page.Name, number)
			}
			for n, slot := range free {
				if offset+n >= len(matches) {
					break
				}
				c := matches[offset+n]
				generated.Keys[slot] = Binding{Control: c.ID, Label: c.Label}
			}
			result.Pages = append(result.Pages, generated)
			if offset+len(free) >= len(matches) {
				break
			}
		}
	}
	return result
}
