package voicemeeter

import (
	"fmt"
	"math"
	"regexp"
	"sound-snoofer/internal/model"
	"strconv"
)

var mixerParameter = regexp.MustCompile(`^(Strip|Bus)\[([0-7])\]\.(Gain|Mute)$`)

// SetMixer accepts only native gain/mute domains; no arbitrary script input.
func (c *Client) SetMixer(param string, v float32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	m := mixerParameter.FindStringSubmatch(param)
	if m == nil || math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
		return fmt.Errorf("invalid mixer parameter")
	}
	if (m[3] == "Gain" && (v < -60 || v > 12)) || (m[3] == "Mute" && v != 0 && v != 1) {
		return fmt.Errorf("invalid mixer value")
	}
	if err := c.refresh(); err != nil {
		return err
	}
	edition, code := c.api.Edition()
	if err := status("mixer edition", code); err != nil {
		return err
	}
	limit := model.StripCount(int(edition))
	if m[1] == "Bus" {
		limit, _ = model.Limits(int(edition))
	}
	index, _ := strconv.Atoi(m[2])
	if index >= limit {
		return fmt.Errorf("mixer target unavailable")
	}
	if m[3] == "Mute" {
		return status("mute", c.api.SetNumber(param, int(v)))
	}
	api, ok := c.api.(interface{ SetScalar(string, float32) int32 })
	if !ok {
		return fmt.Errorf("gain API unavailable")
	}
	return status("gain", api.SetScalar(param, v))
}
func (c *Client) RestartEngine() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.refresh(); err != nil {
		return err
	}
	return status("restart audio engine", c.api.SetNumber("Command.Restart", 1))
}
