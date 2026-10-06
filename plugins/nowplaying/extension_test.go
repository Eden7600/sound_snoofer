package nowplaying

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sound-snoofer/snoofer"
)

func TestSaveExtension(t *testing.T) {
	dir := t.TempDir()
	w := newWorker(snoofer.Services{Path: filepath.Join(dir, "snoofer.json")}, nil, defaultTiming)
	w.settings = Settings{Port: 50000, Token: "abc"}
	if extensionVersion == "" {
		t.Fatal("bundled manifest version unreadable")
	}
	if w.extensionSaved() {
		t.Fatal("saved before saving")
	}
	if err := w.saveExtension(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"manifest.json", "page.js", "bridge.js", "background.js", "logic.js"} {
		if _, err := os.Stat(filepath.Join(dir, "browser-extension", name)); err != nil {
			t.Error(name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "browser-extension", "logic.test.mjs")); err == nil {
		t.Error("tests shipped with the extension")
	}
	config, err := os.ReadFile(filepath.Join(dir, "browser-extension", "config.js"))
	if err != nil || strings.TrimSpace(string(config)) != `export default {"port":50000,"token":"abc"};` {
		t.Fatalf("config.js %q %v", config, err)
	}
	if !w.extensionSaved() {
		t.Fatal("saved files not recognised")
	}
	w.settings.Token = "new"
	if w.extensionSaved() {
		t.Fatal("stale token reported as saved")
	}
}
