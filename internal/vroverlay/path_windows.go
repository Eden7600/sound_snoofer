//go:build windows

package vroverlay

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxPathRegistry = 64 * 1024

func runtimePath(reader io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxPathRegistry+1))
	if err != nil {
		return "", fmt.Errorf("read OpenVR runtime paths: %w", err)
	}
	if len(data) > maxPathRegistry {
		return "", errors.New("OpenVR runtime paths exceed 64 KiB")
	}
	var paths struct {
		Runtime []string `json:"runtime"`
	}
	if err := json.Unmarshal(data, &paths); err != nil {
		return "", fmt.Errorf("parse OpenVR runtime paths: %w", err)
	}
	if len(paths.Runtime) == 0 || strings.IndexByte(paths.Runtime[0], 0) >= 0 || !filepath.IsAbs(paths.Runtime[0]) {
		return "", errors.New("OpenVR runtime path must be an absolute directory")
	}
	// The first entry is active; remaining entries are history, not fallbacks.
	return filepath.Clean(paths.Runtime[0]), nil
}

func installedLoaderPath() (string, error) {
	local := os.Getenv("LOCALAPPDATA")
	if !filepath.IsAbs(local) {
		return "", errors.New("LocalAppData is unavailable for OpenVR discovery")
	}
	f, err := os.Open(filepath.Join(local, "openvr", "openvrpaths.vrpath"))
	if err != nil {
		return "", fmt.Errorf("find installed OpenVR runtime: %w", err)
	}
	path, readErr := runtimePath(f)
	if err = errors.Join(readErr, f.Close()); err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("find OpenVR runtime directory: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("OpenVR runtime path is not a directory")
	}
	return filepath.Join(path, "bin", "win64", "openvr_api.dll"), nil
}
