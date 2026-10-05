package config

import "sound-snoofer/internal/storage"

// WriteJournal durably replaces a private recovery/ownership journal.
func WriteJournal(path string, value any) error   { return storage.WriteJournal(path, value) }
func replaceBytes(path string, data []byte) error { return storage.Replace(path, data) }
