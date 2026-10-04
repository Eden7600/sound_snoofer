package config

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed default.json
var defaultConfig []byte

// EnsureDefault provisions the adjacent configuration without replacing user data.
func EnsureDefault(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("create default config: %w", err)
	}
	_, writeErr := f.Write(defaultConfig)
	closeErr := f.Close()
	if writeErr != nil {
		return fmt.Errorf("write default config: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close default config: %w", closeErr)
	}
	return nil
}
