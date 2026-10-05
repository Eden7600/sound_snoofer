package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteJournal durably replaces a small private recovery/ownership journal.
func WriteJournal(path string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".snoofer-journal-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = replaceState(name, path); err != nil {
		return fmt.Errorf("save journal: %w", err)
	}
	return nil
}
