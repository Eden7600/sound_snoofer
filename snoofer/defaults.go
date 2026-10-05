package snoofer

import (
	"encoding/json"
	"fmt"
)

// Defaults creates a disabled configuration for a new installation.
func Defaults(plugins []Plugin) Config {
	c := Config{Version: 1, Plugins: map[string]PluginConfig{}}
	for _, p := range plugins {
		settings := p.Defaults
		if len(settings) == 0 {
			settings = json.RawMessage("{}")
		}
		c.Plugins[p.ID] = PluginConfig{Settings: settings}
	}
	return c
}

// InitialSettings supplies inert defaults when enabling an omitted compiled plugin.
func (h *Host) initialSettings(id string) json.RawMessage {
	if b := h.plugins[id].Defaults; len(b) > 0 {
		return append(json.RawMessage(nil), b...)
	}
	return json.RawMessage("{}")
}

// MarshalSettings encodes inert default settings; invalid built-in constants are programming errors.
func MarshalSettings(value any) json.RawMessage {
	b, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("invalid static settings: %v", err))
	}
	return b
}
