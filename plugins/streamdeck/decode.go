package streamdeck

import (
	"encoding/json"
	"fmt"

	"sound-snoofer/snoofer"
)

// UnmarshalJSON rejects positions beyond the physical surface rather than
// silently truncating JSON arrays into Go's fixed-size arrays.
func (l *Layout) UnmarshalJSON(data []byte) error {
	type page struct {
		AutoControls string    `json:"auto_controls,omitempty"`
		ID           string    `json:"id"`
		Name         string    `json:"name"`
		Keys         []Binding `json:"keys"`
		Dials        []Binding `json:"dials"`
	}
	var raw struct {
		Home        string    `json:"home"`
		Pages       []page    `json:"pages"`
		SharedKeys  []Binding `json:"shared_keys"`
		SharedDials []Binding `json:"shared_dials"`
	}
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
		v := Page{ID: p.ID, Name: p.Name, AutoControls: p.AutoControls}
		copy(v.Keys[:], p.Keys)
		copy(v.Dials[:], p.Dials)
		next.Pages = append(next.Pages, v)
	}
	*l = next
	return nil
}

var _ json.Unmarshaler = (*Layout)(nil)
