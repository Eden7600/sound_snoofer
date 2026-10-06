package streamdeck

import (
	"fmt"
	"strings"

	"sound-snoofer/snoofer"
)

// Columns is the key grid width; regions are rectangles on this grid.
const Columns = 9

// Region fills a rectangle of keys from a control collection.
type Region struct {
	Source string `json:"source"` // Collection ID, or a legacy ID prefix ending in "-".
	First  int    `json:"first"`  // Zero-based key indexes of opposite corners.
	Last   int    `json:"last"`
}

// cells lists the rectangle's keys in row-major order.
func (r Region) cells() []int {
	top, bottom := min(r.First/Columns, r.Last/Columns), max(r.First/Columns, r.Last/Columns)
	left, right := min(r.First%Columns, r.Last%Columns), max(r.First%Columns, r.Last%Columns)
	var out []int
	for row := top; row <= bottom; row++ {
		for column := left; column <= right; column++ {
			out = append(out, row*Columns+column)
		}
	}
	return out
}

// fill is a region in effect: saved regions, or the whole-page region a
// legacy auto_controls prefix implies.
type fill struct {
	Region
	prefix bool // Match control IDs by prefix rather than by collection.
}

func (f fill) matches(c snoofer.Control) bool {
	if f.prefix {
		return strings.HasPrefix(c.ID, f.Source)
	}
	return c.Collection == f.Source
}

// fills returns the page's regions in effect. A legacy prefix covers every key.
func (p Page) fills() []fill {
	if len(p.Regions) == 0 {
		if p.AutoControls == "" {
			return nil
		}
		return []fill{{Region: Region{Source: p.AutoControls, First: 0, Last: Keys - 1}, prefix: true}}
	}
	out := make([]fill, 0, len(p.Regions))
	for _, r := range p.Regions {
		out = append(out, fill{Region: r, prefix: strings.HasSuffix(r.Source, "-")})
	}
	return out
}

// validateRegions checks a page's saved regions against the layout's shared keys.
func (l Layout) validateRegions(p Page) error {
	if len(p.Regions) == 0 {
		return nil
	}
	if p.AutoControls != "" {
		return fmt.Errorf("%s uses both an automatic prefix and regions", p.Name)
	}
	owner := map[int]int{}
	for n, r := range p.Regions {
		if r.First < 0 || r.First >= Keys || r.Last < 0 || r.Last >= Keys {
			return fmt.Errorf("%s region %d is outside the deck", p.Name, n+1)
		}
		if !strings.Contains(r.Source, ".") {
			return fmt.Errorf("%s region %d needs a provider source", p.Name, n+1)
		}
		free := 0
		for _, key := range r.cells() {
			if other, taken := owner[key]; taken {
				return fmt.Errorf("%s regions %d and %d overlap", p.Name, other+1, n+1)
			}
			owner[key] = n
			if p.Keys[key].Control == "" && l.SharedKeys[key].Control == "" {
				free++
			}
		}
		if free == 0 {
			return fmt.Errorf("%s region %d has no free key", p.Name, n+1)
		}
	}
	return nil
}
