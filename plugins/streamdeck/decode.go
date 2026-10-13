package streamdeck

import (
	"encoding/json"
	"fmt"

	"sound-snoofer/snoofer"
)

// pageJSON and layoutJSON are the saved forms, with positions as lists.
type pageJSON struct {
	AutoControls string    `json:"auto_controls,omitempty"`
	Regions      []Region  `json:"regions,omitempty"`
	DialRegions  []Region  `json:"dial_regions,omitempty"`
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Keys         []Binding `json:"keys"`
	Dials        []Binding `json:"dials"`
}
type layoutJSON struct {
	Home        string     `json:"home"`
	Pages       []pageJSON `json:"pages"`
	SharedKeys  []Binding  `json:"shared_keys"`
	SharedDials []Binding  `json:"shared_dials"`
}

// UnmarshalJSON rejects positions beyond the largest surface rather than
// silently truncating JSON arrays into Go's fixed-size arrays. The layout's
// model is set by where it is stored, and Validate checks its geometry.
func (l *Layout) UnmarshalJSON(data []byte) error {
	var raw layoutJSON
	if err := snoofer.DecodeSettings(data, &raw); err != nil {
		return err
	}
	if len(raw.SharedKeys) > Keys || len(raw.SharedDials) > Dials {
		return fmt.Errorf("shared positions exceed the device; dial 6 is reserved")
	}
	next := Layout{Home: raw.Home}
	copy(next.SharedKeys[:], raw.SharedKeys)
	copy(next.SharedDials[:], raw.SharedDials)
	for _, p := range raw.Pages {
		if len(p.Keys) > Keys || len(p.Dials) > Dials {
			return fmt.Errorf("page %s exceeds device positions; dial 6 is reserved", p.Name)
		}
		v := Page{ID: p.ID, Name: p.Name, AutoControls: p.AutoControls, Regions: p.Regions, DialRegions: p.DialRegions}
		copy(v.Keys[:], p.Keys)
		copy(v.Dials[:], p.Dials)
		next.Pages = append(next.Pages, v)
	}
	*l = next
	return nil
}

// MarshalJSON saves only the positions the layout's geometry has.
func (l Layout) MarshalJSON() ([]byte, error) {
	sh := l.shape()
	raw := layoutJSON{Home: l.Home, SharedKeys: l.SharedKeys[:sh.keys], SharedDials: l.SharedDials[:sh.dials]}
	for _, p := range l.Pages {
		raw.Pages = append(raw.Pages, pageJSON{AutoControls: p.AutoControls, Regions: p.Regions, DialRegions: p.DialRegions, ID: p.ID, Name: p.Name,
			Keys: p.Keys[:sh.keys], Dials: p.Dials[:sh.dials]})
	}
	return json.Marshal(raw)
}

var _ json.Unmarshaler = (*Layout)(nil)
var _ json.Marshaler = Layout{}
