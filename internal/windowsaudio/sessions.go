package windowsaudio

import (
	"path/filepath"
	"strings"
)

// SystemSounds is the path reported for the Windows system sounds session.
const SystemSounds = "system"

// Session is one Windows audio session on an active render device.
type Session struct {
	Key    string // Device ID and session instance; stable while the session lives.
	Device string // Device friendly name, for diagnostics.
	PID    uint32
	Path   string // Executable path, or SystemSounds.
	Name   string // Program name: version description, plain session name or file name.
	Icon   string // Base64 PNG thumbnail of the executable icon, or empty.
	Active bool
	Volume float64 // 0…1.
	Muted  bool
}

// SessionBackend reads and writes Windows audio sessions. Implementations
// are not safe for concurrent use; the native one belongs to a locked COM
// thread.
type SessionBackend interface {
	// List rescans every session; keys from an earlier list stay valid only
	// if listed again.
	List() ([]Session, error)
	// Peak is the current linear peak (0…1) of a listed session.
	Peak(key string) (float64, error)
	SetVolume(key string, volume float64) error
	SetMute(key string, muted bool) error
	Close()
}

// programName chooses a readable name. Session display names are used only
// when they are plain text, not resource references such as "@%SystemRoot%…"
// or "ms-resource:…".
func programName(path, description, display string) string {
	if path == SystemSounds {
		return "System sounds"
	}
	if d := strings.TrimSpace(description); d != "" {
		return d
	}
	if d := strings.TrimSpace(display); d != "" && !strings.HasPrefix(d, "@") && !strings.HasPrefix(strings.ToLower(d), "ms-resource:") {
		return d
	}
	base := filepath.Base(path)
	if base == "." || base == string(filepath.Separator) {
		return "Unknown app"
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}
