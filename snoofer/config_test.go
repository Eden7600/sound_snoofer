package snoofer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpaqueAndAtomicConfig(t *testing.T) {
	data := []byte(`{"version":1,"plugins":{"absent":{"enabled":false,"settings":{"arbitrary":[1,2]}}}}`)
	c, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	rev, err := Save(path, c, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Save(path, c, "missing"); err == nil {
		t.Fatal("stale save")
	}
	loaded, got, err := Load(path)
	if err != nil || got != rev {
		t.Fatal(got, err)
	}
	if string(loaded.Plugins["absent"].Settings) == "" {
		t.Fatal("lost settings")
	}
	before, _ := os.ReadFile(path)
	c.Version = 2
	if _, err = Save(path, c, rev); err == nil {
		t.Fatal("invalid save")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("partial save")
	}
	for _, bad := range []string{`{"version":1,"version":1,"plugins":{}}`, `{"version":1,"plugins":{},"unknown":1}`, `{"version":1,"plugins":{}} {}`} {
		if _, err := Decode([]byte(bad)); err == nil {
			t.Fatal(bad)
		}
	}
}
