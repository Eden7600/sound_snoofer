package snoofer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"sound-snoofer/internal/ownership"
	"sound-snoofer/internal/storage"
)

// PluginConfig retains opaque settings even when its plugin is absent.
type PluginConfig struct {
	Enabled  bool            `json:"enabled"`
	Settings json.RawMessage `json:"settings"`
}

// Config is the domain-neutral configuration envelope.
type Config struct {
	Version int                     `json:"version"`
	Plugins map[string]PluginConfig `json:"plugins"`
}

// Clone returns a fully independent configuration.
func (c Config) Clone() Config {
	n := Config{Version: c.Version, Plugins: map[string]PluginConfig{}}
	for id, p := range c.Plugins {
		p.Settings = append(json.RawMessage(nil), p.Settings...)
		n.Plugins[id] = p
	}
	return n
}

// Decode validates only the envelope; plugin settings remain opaque.
func Decode(data []byte) (Config, error) {
	var c Config
	if err := storage.Decode(data, &c); err != nil {
		return c, err
	}
	if c.Version != 1 || c.Plugins == nil {
		return c, fmt.Errorf("expected Snoofer version 1 and plugins object")
	}
	for id, p := range c.Plugins {
		if !validID(id) || len(p.Settings) == 0 || string(p.Settings) == "null" {
			return c, fmt.Errorf("%s requires settings object", id)
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(p.Settings, &object); err != nil || object == nil {
			return c, fmt.Errorf("%s settings must be an object", id)
		}
	}
	return c, nil
}

// Load reads a config and its revision token.
func Load(path string) (Config, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, "", err
	}
	c, err := Decode(b)
	return c, token(b), err
}
func token(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }

// Save validates and atomically replaces a config only if the revision still matches.
func Save(path string, c Config, expected string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	release, err := ownership.AcquireState(absolute)
	if err != nil {
		return "", err
	}
	defer release()
	b, err := os.ReadFile(absolute)
	actual := "missing"
	if err == nil {
		actual = token(b)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if actual != expected {
		return "", fmt.Errorf("configuration changed; reload before saving")
	}
	b, err = json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	if _, err = Decode(b); err != nil {
		return "", err
	}
	b = append(b, '\n')
	if err = storage.Replace(absolute, b); err != nil {
		return "", err
	}
	return token(b), nil
}
