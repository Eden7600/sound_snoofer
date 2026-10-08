package streamdeck

import (
	"encoding/json"
	"fmt"

	"sound-snoofer/snoofer"
)

func validateSettings(raw json.RawMessage) error {
	var settings Settings
	if err := snoofer.DecodeSettings(raw, &settings); err != nil {
		return err
	}
	if b := settings.Brightness; b != nil && (*b < 1 || *b > 100) {
		return fmt.Errorf("brightness must be 1-100")
	}
	if err := settings.Layout.Validate(nil); err != nil {
		return err
	}
	for serial, l := range settings.Serials {
		if serial == "" {
			return fmt.Errorf("serial must be nonempty")
		}
		if err := l.Validate(nil); err != nil {
			return err
		}
	}
	return nil
}
