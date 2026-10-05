package snoofer

import (
	"encoding/json"
	"fmt"
	"reflect"

	"sound-snoofer/internal/storage"
)

// DecodeSettings validates an enabled plugin's settings.
func DecodeSettings(data []byte, value any) error { return storage.Decode(data, value) }

func (h *Host) saveSettings(id string, expected, next json.RawMessage) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.stopping {
		return fmt.Errorf("host is stopping")
	}
	current, rev, err := Load(h.services.Path)
	if err != nil {
		return err
	}
	var a, b any
	if err = json.Unmarshal(current.Plugins[id].Settings, &a); err != nil {
		return err
	}
	if err = json.Unmarshal(expected, &b); err != nil {
		return err
	}
	if !reflect.DeepEqual(a, b) {
		return fmt.Errorf("plugin settings changed; reload before saving")
	}
	p := current.Plugins[id]
	p.Settings = next
	current.Plugins[id] = p
	if _, err = Save(h.services.Path, current, rev); err != nil {
		return err
	}
	h.config = current
	return nil
}
