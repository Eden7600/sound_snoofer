// Package streamdeck provides a semantic control surface and its integrated editor.
package streamdeck

import (
	"fmt"
	"slices"
	"strings"

	"sound-snoofer/snoofer"
)

const Keys = 36
const Dials = 5

// Binding persists identity and a label even when the provider is unavailable.
type Binding struct {
	Control string `json:"control"`
	Label   string `json:"label"`
}

// Page covers keys and the five user-assignable dials.
type Page struct {
	AutoControls string         `json:"auto_controls,omitempty"`
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Keys         [Keys]Binding  `json:"keys"`
	Dials        [Dials]Binding `json:"dials"`
}

// Layout reserves shared positions globally and the sixth dial for navigation.
type Layout struct {
	Home        string         `json:"home"`
	Pages       []Page         `json:"pages"`
	SharedKeys  [Keys]Binding  `json:"shared_keys"`
	SharedDials [Dials]Binding `json:"shared_dials"`
}

// Settings contains a default layout and optional exact serial overrides.
type Settings struct {
	Layout  Layout            `json:"layout"`
	Serials map[string]Layout `json:"serials,omitempty"`
}

func (l Layout) clone() Layout { l.Pages = slices.Clone(l.Pages); return l }
func (l Layout) index(id string) int {
	for n, p := range l.Pages {
		if p.ID == id {
			return n
		}
	}
	return -1
}
func (l Layout) effective(id string) Page {
	n := l.index(id)
	if n < 0 {
		n = l.index(l.Home)
	}
	if n < 0 {
		return Page{}
	}
	p := l.Pages[n]
	for n, b := range l.SharedKeys {
		if b.Control != "" {
			p.Keys[n] = b
		}
	}
	for n, b := range l.SharedDials {
		if b.Control != "" {
			p.Dials[n] = b
		}
	}
	return p
}
func (l Layout) next(id string, delta int) string {
	n := l.index(id)
	if n < 0 {
		n = l.index(l.Home)
	}
	if len(l.Pages) == 0 {
		return ""
	}
	n = ((n+delta)%len(l.Pages) + len(l.Pages)) % len(l.Pages)
	return l.Pages[n].ID
}
func (l *Layout) remove(id string) error {
	if len(l.Pages) <= 1 {
		return fmt.Errorf("keep at least one page")
	}
	n := l.index(id)
	if n < 0 {
		return fmt.Errorf("unknown page")
	}
	l.Pages = slices.Delete(l.Pages, n, n+1)
	if l.Home == id {
		l.Home = l.Pages[0].ID
	}
	return nil
}

// Validate rejects collisions and incompatible controls; missing providers remain valid.
func (l Layout) Validate(controls []snoofer.Control) error {
	if len(l.Pages) == 0 || l.index(l.Home) < 0 {
		return fmt.Errorf("layout needs pages and a valid Home")
	}
	seen := map[string]bool{}
	check := func(b Binding, dial bool) error {
		if b.Control == "" {
			return nil
		}
		for _, c := range controls {
			if c.ID == b.Control {
				op := "press"
				if dial {
					op = "adjust"
				}
				if !slices.Contains(c.Operations, op) && !(!dial && slices.Contains(c.Operations, "set")) {
					return fmt.Errorf("%s does not support this position", c.Label)
				}
			}
		}
		return nil
	}
	for _, p := range l.Pages {
		if p.ID == "" || p.Name == "" || seen[p.ID] {
			return fmt.Errorf("page IDs must be unique and names nonempty")
		}
		seen[p.ID] = true
		if strings.Contains(p.ID, "~auto~") {
			return fmt.Errorf("page ID uses reserved automatic suffix")
		}
		if p.AutoControls != "" {
			if !strings.Contains(p.AutoControls, ".") {
				return fmt.Errorf("automatic controls need a provider prefix")
			}
			free := 0
			for n, b := range p.Keys {
				if b.Control == "" && l.SharedKeys[n].Control == "" {
					free++
				}
			}
			if free == 0 {
				return fmt.Errorf("automatic page needs a free key")
			}
		}
		for n, b := range p.Keys {
			if b.Control != "" && l.SharedKeys[n].Control != "" {
				return fmt.Errorf("%s key %d conflicts with shared binding", p.Name, n+1)
			}
			if err := check(b, false); err != nil {
				return err
			}
		}
		for n, b := range p.Dials {
			if b.Control != "" && l.SharedDials[n].Control != "" {
				return fmt.Errorf("%s dial %d conflicts with shared binding", p.Name, n+1)
			}
			if err := check(b, true); err != nil {
				return err
			}
		}
	}
	for _, b := range l.SharedKeys {
		if err := check(b, false); err != nil {
			return err
		}
	}
	for _, b := range l.SharedDials {
		if err := check(b, true); err != nil {
			return err
		}
	}
	return nil
}

// pageNames follows the same cyclic order as navigation, including automatic pages.
func (l Layout) pageNames(id string) [3]string {
	current := l.effective(id).ID
	return [3]string{l.effective(l.next(current, -1)).Name, l.effective(current).Name, l.effective(l.next(current, 1)).Name}
}
