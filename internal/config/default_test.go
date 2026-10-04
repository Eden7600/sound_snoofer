package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPreservesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := EnsureDefault(path); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("user data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDefault(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "user data" {
		t.Fatal(string(got), err)
	}
	if err := EnsureDefault(filepath.Join(path, "bad.json")); err == nil {
		t.Fatal("unwritable path accepted")
	}
}
