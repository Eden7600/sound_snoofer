package nowplaying

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// extensionFiles is the browser extension, saved for Load unpacked.
// config.js is written alongside with the port and token.
//
//go:embed extension/manifest.json extension/*.js
var extensionFiles embed.FS

// extensionVersion is the bundled extension's manifest version.
var extensionVersion = func() string {
	data, err := extensionFiles.ReadFile("extension/manifest.json")
	if err != nil {
		return ""
	}
	var m struct{ Version string }
	if json.Unmarshal(data, &m) != nil {
		return ""
	}
	return m.Version
}()

// extensionDir is where the extension is saved: beside the configuration,
// like other generated data.
func (w *worker) extensionDir() string {
	if w.services.Path == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(w.services.Path), "browser-extension")
}

// extensionConfig is config.js for the current port and token.
func (w *worker) extensionConfig() []byte {
	data, _ := json.Marshal(map[string]any{"port": w.settings.port(), "token": w.settings.Token}) // Plain values always marshal.
	return []byte("export default " + string(data) + ";\n")
}

// saveExtension writes the bundled files and config.js.
func (w *worker) saveExtension() error {
	dir := w.extensionDir()
	if dir == "" {
		return errors.New("no configuration folder to save beside")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	err := fs.WalkDir(extensionFiles, "extension", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := extensionFiles.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, d.Name()), data, 0o644)
	})
	if err != nil {
		return fmt.Errorf("save extension: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.js"), w.extensionConfig(), 0o644); err != nil {
		return fmt.Errorf("save extension: %w", err)
	}
	return nil
}

// extensionSaved reports whether the saved files match this version, port
// and token. It reads the folder, so callers check it only after changes.
func (w *worker) extensionSaved() bool {
	dir := w.extensionDir()
	if dir == "" {
		return false
	}
	config, err := os.ReadFile(filepath.Join(dir, "config.js"))
	if err != nil || string(config) != string(w.extensionConfig()) {
		return false
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	bundled, _ := extensionFiles.ReadFile("extension/manifest.json")
	return err == nil && string(manifest) == string(bundled)
}
