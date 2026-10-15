// Package streamdeck provides a semantic control surface and its integrated editor.
package streamdeck

import (
	"fmt"
	"slices"
	"strings"

	"sound-snoofer/snoofer"
)

// Keys and Dials are the most keys and user dials of any model; a page holds
// this many and a layout's geometry decides how many it uses.
const Keys = 36
const Dials = 5

// Binding persists identity and a label even when the provider is unavailable.
type Binding struct {
	Control string `json:"control"`
	Label   string `json:"label"`
}

// Page covers keys and the five user-assignable dials. Regions fill keys and
// DialRegions fill dials from control collections; AutoControls is the
// legacy whole-page prefix.
type Page struct {
	AutoControls string         `json:"auto_controls,omitempty"`
	Regions      []Region       `json:"regions,omitempty"`
	DialRegions  []Region       `json:"dial_regions,omitempty"` // First and Last are dial indexes 0–4.
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Keys         [Keys]Binding  `json:"keys"`
	Dials        [Dials]Binding `json:"dials"`
}

// Layout reserves shared positions globally and the last dial (or the Pages
// key) for navigation.
type Layout struct {
	Home        string         `json:"home"`
	Pages       []Page         `json:"pages"`
	SharedKeys  [Keys]Binding  `json:"shared_keys"`
	SharedDials [Dials]Binding `json:"shared_dials"`
	model       string         // Model ID, from where the layout is stored; empty is the + XL.
}

// Settings holds a layout per model and optional per-deck layouts. Layout is
// the + XL's; Models holds the others by model ID, and a model without one
// uses its shipped default. Serials are decks' own layouts, of the model
// their Devices record names (the + XL without one).
type Settings struct {
	Layout  Layout            `json:"layout"`
	Models  map[string]Layout `json:"models,omitempty"`
	Serials map[string]Layout `json:"serials,omitempty"`
	// Devices remembers decks by serial.
	Devices map[string]Deck `json:"devices,omitempty"`
	// Paging is "independent" (the default when empty) or "mirrored".
	Paging string `json:"paging,omitempty"`
	// Brightness is the backlight (1-100) when the deck is not dark; nil
	// means 100.
	Brightness *int `json:"brightness,omitempty"`
}

// Deck is a remembered deck: its model and an optional name.
type Deck struct {
	Model string `json:"model"`
	Name  string `json:"name,omitempty"`
}

// Paging modes.
const (
	pagingIndependent = "independent"
	pagingMirrored    = "mirrored"
)

// backlight is the validated brightness.
func (s Settings) backlight() int {
	if s.Brightness == nil {
		return 100
	}
	return *s.Brightness
}

// clone copies pages deeply enough for draft edits, including region lists.
func (l Layout) clone() Layout {
	l.Pages = slices.Clone(l.Pages)
	for n := range l.Pages {
		l.Pages[n].Regions = slices.Clone(l.Pages[n].Regions)
		for r := range l.Pages[n].Regions {
			l.Pages[n].Regions[r].Sources = slices.Clone(l.Pages[n].Regions[r].Sources)
		}
		l.Pages[n].DialRegions = slices.Clone(l.Pages[n].DialRegions)
	}
	return l
}
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

// next is the page dial order. A page that binds a scroll key is one stop:
// its overflow sets are skipped, and leaving any of them counts from the page.
func (l Layout) next(id string, delta int) string {
	var stops []string
	for _, p := range l.Pages {
		if base, _, overflow := strings.Cut(p.ID, "~auto~"); overflow && l.scrolls(base) {
			continue
		}
		stops = append(stops, p.ID)
	}
	if len(stops) == 0 {
		return ""
	}
	if base, _, overflow := strings.Cut(id, "~auto~"); overflow && l.scrolls(base) {
		id = base
	}
	n := slices.Index(stops, id)
	if n < 0 {
		n = max(0, slices.Index(stops, l.Home))
	}
	n = ((n+delta)%len(stops) + len(stops)) % len(stops)
	return stops[n]
}

// scrolls reports whether a page binds a scroll key.
func (l Layout) scrolls(id string) bool {
	for _, b := range l.effective(id).Keys {
		if strings.HasPrefix(b.Control, scrollPrefix) {
			return true
		}
	}
	return false
}

// sets lists the shown page and its overflow sets in order.
func (l Layout) sets(shown string) []string {
	base, _, _ := strings.Cut(shown, "~auto~")
	var out []string
	for _, p := range l.Pages {
		if p.ID == base || strings.HasPrefix(p.ID, base+"~auto~") {
			out = append(out, p.ID)
		}
	}
	return out
}

// scroll returns the set delta steps from shown, wrapping at either end.
func (l Layout) scroll(shown string, delta int) string {
	sets := l.sets(shown)
	n := slices.Index(sets, shown)
	if n < 0 {
		return shown
	}
	return sets[((n+delta)%len(sets)+len(sets))%len(sets)]
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

// bindingCompatible is shared by layout validation and editor choices.
// Status controls occupy keys for display only, regardless of advertised actions.
func bindingCompatible(c snoofer.Control, dial bool) bool {
	if c.Kind == "text" {
		return false
	}
	if c.Kind == "status" {
		return !dial
	}
	if dial {
		return slices.Contains(c.Operations, "adjust")
	}
	return slices.Contains(c.Operations, "press") || slices.Contains(c.Operations, "set")
}

// Validate rejects collisions and incompatible controls; missing providers remain valid.
func (l Layout) Validate(controls []snoofer.Control) error {
	if len(l.Pages) == 0 || l.index(l.Home) < 0 {
		return fmt.Errorf("layout needs pages and a valid Home")
	}
	seen := map[string]bool{}
	sh := l.shape()
	// placed rejects a binding at a position the geometry lacks or reserves.
	placed := func(where string, n int, b Binding, dial bool) error {
		switch {
		case b.Control == "":
			return nil
		case dial && n >= sh.dials:
			return fmt.Errorf("%s dial %d is not on this deck", where, n+1)
		case !dial && n == sh.pagesKey:
			return fmt.Errorf("%s key %d is the Pages key", where, n+1)
		case !dial && n >= sh.keys:
			return fmt.Errorf("%s key %d is not on this deck", where, n+1)
		}
		return nil
	}
	check := func(b Binding, dial bool) error {
		if b.Control == "" {
			return nil
		}
		for _, c := range controls {
			if c.ID == b.Control {
				if c.Kind == "text" {
					return fmt.Errorf("%s needs text entry and has no deck position", c.Label)
				}
				if !bindingCompatible(c, dial) {
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
		if err := l.validateRegions(p); err != nil {
			return err
		}
		if p.AutoControls != "" {
			if !strings.Contains(p.AutoControls, ".") {
				return fmt.Errorf("automatic controls need a provider prefix")
			}
			free := 0
			for n, b := range p.Keys {
				if sh.usable(n) && b.Control == "" && l.SharedKeys[n].Control == "" {
					free++
				}
			}
			if free == 0 {
				return fmt.Errorf("automatic page needs a free key")
			}
		}
		for n, b := range p.Keys {
			if err := placed(p.Name, n, b, false); err != nil {
				return err
			}
			if b.Control != "" && l.SharedKeys[n].Control != "" {
				return fmt.Errorf("%s key %d conflicts with shared binding", p.Name, n+1)
			}
			if err := check(b, false); err != nil {
				return err
			}
		}
		for n, b := range p.Dials {
			if err := placed(p.Name, n, b, true); err != nil {
				return err
			}
			if b.Control != "" && l.SharedDials[n].Control != "" {
				return fmt.Errorf("%s dial %d conflicts with shared binding", p.Name, n+1)
			}
			if err := check(b, true); err != nil {
				return err
			}
		}
	}
	for n, b := range l.SharedKeys {
		if err := placed("Shared", n, b, false); err != nil {
			return err
		}
		if err := check(b, false); err != nil {
			return err
		}
	}
	for n, b := range l.SharedDials {
		if err := placed("Shared", n, b, true); err != nil {
			return err
		}
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

// resetFocusID is the key that returns every focus dial to its default.
const resetFocusID = "streamdeck.reset-focus"

// resetFocusControl is the Reset focus key; a press sends reset to every
// control that offers it.
func resetFocusControl() snoofer.Control {
	return snoofer.Control{ID: resetFocusID, Label: "Reset Focus", ShortLabel: "Auto", Group: "Stream Deck Pages", Kind: "command", Icon: "focus-reset",
		Operations: []string{"press"}, Available: true}
}

// scrollPrefix identifies the keys that page through overflow sets.
const scrollPrefix = "streamdeck.scroll-"

// scrollControls describes the expanded page being drawn: its set position,
// or Hidden when it has a single set.
func scrollControls(l Layout, shown string) []snoofer.Control {
	sets := l.sets(shown)
	value := fmt.Sprintf("%d/%d", slices.Index(sets, shown)+1, len(sets))
	out := []snoofer.Control{}
	for _, key := range []struct{ id, label, short string }{{"up", "Scroll Up", "Up"}, {"down", "Scroll Down", "Down"}} {
		out = append(out, snoofer.Control{ID: scrollPrefix + key.id, Label: key.label, ShortLabel: key.short, Group: "Stream Deck Pages",
			Kind: "command", Icon: "deck-" + key.id, Value: value, Hidden: len(sets) < 2, Operations: []string{"press"}, Available: true})
	}
	return out
}

// scrollDelta is the set step of a scroll key.
func scrollDelta(id string) int {
	if id == scrollPrefix+"up" {
		return -1
	}
	return 1
}

// gotoPrefix identifies go-to page controls; the page ID follows it.
const gotoPrefix = "streamdeck.goto-"

// gotoControls offers a key per saved page except Home, which the page
// dial's press already reaches. Here marks the shown page, including its
// automatic overflow pages.
func gotoControls(l Layout, shown string) []snoofer.Control {
	base, _, _ := strings.Cut(shown, "~auto~")
	out := make([]snoofer.Control, 0, len(l.Pages))
	for _, p := range l.Pages {
		if p.ID == l.Home {
			continue
		}
		value := ""
		if p.ID == base {
			value = "Here"
		}
		out = append(out, snoofer.Control{ID: gotoPrefix + p.ID, Label: "Go to " + p.Name, ShortLabel: p.Name, Group: "Stream Deck Pages",
			Kind: "command", Icon: "deck-page", Value: value, Operations: []string{"press"}, Available: true})
	}
	return out
}
