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
	if step := settings.GainStepDB; step != nil && (*step <= 0 || *step > 6) {
		return fmt.Errorf("gain_step_db must be greater than 0 and at most 6")
	}
	if c.VR != nil {
		return fmt.Errorf("VR belongs in its plugin settings")
	}
	return nil
}
