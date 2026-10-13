package streamdeck

import (
	"encoding/json"
	"fmt"
	"maps"

	device "sound-snoofer/internal/streamdeck"
	"sound-snoofer/snoofer"
)

func validateSettings(raw json.RawMessage) error {
	settings, err := decodeSettings(raw)
	if err != nil {
		return err
	}
	return settings.validate(nil)
}

// decodeSettings decodes settings and gives each stored layout its model.
func decodeSettings(raw json.RawMessage) (Settings, error) {
	var settings Settings
	if err := snoofer.DecodeSettings(raw, &settings); err != nil {
		return Settings{}, err
	}
	if b := settings.Brightness; b != nil && (*b < 1 || *b > 100) {
		return Settings{}, fmt.Errorf("brightness must be 1-100")
	}
	switch settings.Paging {
	case "", pagingIndependent, pagingMirrored:
	default:
		return Settings{}, fmt.Errorf("paging must be independent or mirrored")
	}
	settings.Layout = settings.Layout.forModel(plusXLModel)
	for id, l := range settings.Models {
		if _, ok := device.ModelByID(id); !ok || id == plusXLModel {
			return Settings{}, fmt.Errorf("models: unknown Stream Deck model %q", id)
		}
		settings.Models[id] = l.forModel(id)
	}
	for serial, d := range settings.Devices {
		if serial == "" {
			return Settings{}, fmt.Errorf("serial must be nonempty")
		}
		if _, ok := device.ModelByID(d.Model); !ok {
			return Settings{}, fmt.Errorf("deck %s: unknown Stream Deck model %q", serial, d.Model)
		}
	}
	for serial, l := range settings.Serials {
		if serial == "" {
			return Settings{}, fmt.Errorf("serial must be nonempty")
		}
		settings.Serials[serial] = l.forModel(settings.Devices[serial].Model)
	}
	return settings, nil
}

// validate checks every stored layout against controls; missing providers
// remain valid.
func (s Settings) validate(controls []snoofer.Control) error {
	if err := s.Layout.Validate(controls); err != nil {
		return err
	}
	for id, l := range s.Models {
		if err := l.Validate(controls); err != nil {
			return fmt.Errorf("%s layout: %w", id, err)
		}
	}
	for serial, l := range s.Serials {
		if err := l.Validate(controls); err != nil {
			return fmt.Errorf("deck %s layout: %w", serial, err)
		}
	}
	return nil
}

// modelLayout is a model's layout: saved, or its shipped default.
func (s Settings) modelLayout(model string) Layout {
	if model == "" || model == plusXLModel {
		return s.Layout
	}
	if l, ok := s.Models[model]; ok {
		return l
	}
	return DefaultLayoutFor(model)
}

// layoutFor is the layout a deck uses: its own when it is of the deck's
// model, else its model's.
func (s Settings) layoutFor(model, serial string) Layout {
	if l, ok := s.Serials[serial]; ok && serial != "" && l.model == modelKey(model) {
		return l
	}
	return s.modelLayout(model)
}

// clone copies the settings' maps so a change can be saved before it applies.
func (s Settings) clone() Settings {
	s.Models = maps.Clone(s.Models)
	s.Serials = maps.Clone(s.Serials)
	s.Devices = maps.Clone(s.Devices)
	if s.Models == nil {
		s.Models = map[string]Layout{}
	}
	if s.Serials == nil {
		s.Serials = map[string]Layout{}
	}
	if s.Devices == nil {
		s.Devices = map[string]Deck{}
	}
	return s
}
