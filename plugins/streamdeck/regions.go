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

// Region fills a rectangle of keys from a control collection. Sources, when
// set, are further collections that share the rectangle's rows after Source,
// each taking rows by need, so a source with nothing to show leaves its rows
// to the others.
type Region struct {
	Source  string   `json:"source"` // Collection ID, or a legacy ID prefix ending in "-".
	Sources []string `json:"sources,omitempty"`
	First   int      `json:"first"` // Zero-based key indexes of opposite corners.
	Last    int      `json:"last"`
	Clip    bool     `json:"clip,omitempty"` // Show only what fits; never add overflow sets.
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

// rows lists the rectangle's keys row by row.
func (r Region) rows() [][]int {
	var out [][]int
	for _, cell := range r.cells() {
		if len(out) == 0 || out[len(out)-1][0]/Columns != cell/Columns {
			out = append(out, nil)
		}
		out[len(out)-1] = append(out[len(out)-1], cell)
	}
	return out
}

// dialCells lists a dial region's dials in order.
func (r Region) dialCells() []int {
	var out []int
	for dial := min(r.First, r.Last); dial <= max(r.First, r.Last); dial++ {
		out = append(out, dial)
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

// validateRegions checks a page's saved regions against the layout's shared
// keys and dials.
func (l Layout) validateRegions(p Page) error {
	if err := l.validateDialRegions(p); err != nil {
		return err
	}
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
		for _, source := range r.Sources {
			if !strings.Contains(source, ".") || strings.HasSuffix(source, "-") || source == r.Source {
				return fmt.Errorf("%s region %d shares rows with an invalid source %q", p.Name, n+1, source)
			}
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

func (l Layout) validateDialRegions(p Page) error {
	owner := map[int]int{}
	for n, r := range p.DialRegions {
		if r.First < 0 || r.First >= Dials || r.Last < 0 || r.Last >= Dials {
			return fmt.Errorf("%s dial region %d is outside the deck", p.Name, n+1)
		}
		if !strings.Contains(r.Source, ".") || strings.HasSuffix(r.Source, "-") {
			return fmt.Errorf("%s dial region %d needs a collection source", p.Name, n+1)
		}
		free := 0
		for _, dial := range r.dialCells() {
			if other, taken := owner[dial]; taken {
				return fmt.Errorf("%s dial regions %d and %d overlap", p.Name, other+1, n+1)
			}
			owner[dial] = n
			if p.Dials[dial].Control == "" && l.SharedDials[dial].Control == "" {
				free++
			}
		}
		if free == 0 {
			return fmt.Errorf("%s dial region %d has no free dial", p.Name, n+1)
		}
	}
	return nil
}

// editRegion applies an editor request to a copy of page n's regions:
// "add" takes "first,last,source", "source" takes "index,source" and
// "remove" takes "index". Positions are editor slots: keys 0–35, then dials
// from Keys. Indexes count key regions first, then dial regions. Editing a
// legacy page first converts its prefix into an explicit whole-page region.
// The caller validates the result.
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
		if err != nil || i < 0 || i >= len(page.Regions)+len(page.DialRegions) {
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
		source := strings.TrimSpace(parts[2])
		switch {
		case first >= Keys && last >= Keys:
			page.DialRegions = append(page.DialRegions, Region{Source: source, First: first - Keys, Last: last - Keys})
		case first < Keys && last < Keys:
			page.Regions = append(page.Regions, Region{Source: source, First: first, Last: last})
		default:
			return l, fmt.Errorf("a region holds keys or dials, not both")
		}
	case "source":
		i, err := index()
		if err != nil || len(parts) < 2 {
			return l, fmt.Errorf("unknown region")
		}
		source := strings.TrimSpace(strings.Join(parts[1:], ","))
		if i < len(page.Regions) {
			page.Regions[i].Source = source
		} else {
			page.DialRegions[i-len(page.Regions)].Source = source
		}
	case "remove":
		i, err := index()
		if err != nil {
			return l, err
		}
		if i < len(page.Regions) {
			page.Regions = slices.Delete(page.Regions, i, i+1)
		} else {
			page.DialRegions = slices.Delete(page.DialRegions, i-len(page.Regions), i-len(page.Regions)+1)
		}
	default:
		return l, fmt.Errorf("unknown region edit %q", op)
	}
	return next, nil
}
