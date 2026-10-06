//go:build !windows

package windowsaudio

import "errors"

// OpenSessions is unavailable outside Windows.
func OpenSessions() (SessionBackend, error) {
	return nil, errors.New("Windows audio sessions unsupported")
}
