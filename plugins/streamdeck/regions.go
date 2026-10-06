package streamdeck

import (
	"fmt"
	"slices"
	"strconv"
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

// editRegion applies an editor request to a copy of page n's regions:
// "add" takes "first,last,source", "source" takes "index,source" and
// "remove" takes "index". Editing a legacy page first converts its prefix
// into an explicit whole-page region. The caller validates the result.
func (l Layout) editRegion(n int, op, value string) (Layout, error) {
	next := l.clone()
	page := &next.Pages[n]
	if len(page.Regions) == 0 && page.AutoControls != "" {
		page.Regions = []Region{{Source: page.AutoControls, First: 0, Last: Keys - 1}}
		page.AutoControls = ""
	}
	parts := strings.SplitN(value, ",", 3)
	index := func() (int, error) {
		i, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || i < 0 || i >= len(page.Regions) {
			return 0, fmt.Errorf("unknown region")
		}
		return i, nil
	}
	switch op {
	case "add":
		if len(parts) != 3 {
			return l, fmt.Errorf("region needs keys and a source")
		}
		first, errFirst := strconv.Atoi(strings.TrimSpace(parts[0]))
		last, errLast := strconv.Atoi(strings.TrimSpace(parts[1]))
		if errFirst != nil || errLast != nil {
			return l, fmt.Errorf("region keys must be numbers")
		}
		page.Regions = append(page.Regions, Region{Source: strings.TrimSpace(parts[2]), First: first, Last: last})
	case "source":
		i, err := index()
		if err != nil || len(parts) < 2 {
			return l, fmt.Errorf("unknown region")
		}
		page.Regions[i].Source = strings.TrimSpace(strings.Join(parts[1:], ","))
	case "remove":
		i, err := index()
		if err != nil {
			return l, err
		}
		page.Regions = slices.Delete(page.Regions, i, i+1)
	default:
		return l, fmt.Errorf("unknown region edit %q", op)
	}
	return next, nil
}
