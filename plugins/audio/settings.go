package audio

import (
	"encoding/json"
	"fmt"

	"sound-snoofer/internal/config"
	"sound-snoofer/snoofer"
)

func validateSettings(raw json.RawMessage) error {
	var settings Settings
	if err := snoofer.DecodeSettings(raw, &settings); err != nil {
		return err
	}
	c, err := config.Decode(settings.Config)
	if err != nil {
		return err
	}
	if err := validateSoundboard(settings.SoundboardInput, c); err != nil {
		return err
	}
	if settings.StatePath == "" {
		return fmt.Errorf("state_path is required")
	}
	if c.VR != nil || c.StreamDeck != nil {
		return fmt.Errorf("VR and Stream Deck belong in their plugin settings")
	}
	return nil
}
