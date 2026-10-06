//go:build windows

package mediasessions

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Opt-in read-only probe: SNOOFER_MEDIA_PROBE names the built companion DLL.
// It lists sessions and fetches artwork; it never sends a command.
func TestNativeSessions(t *testing.T) {
	path := os.Getenv("SNOOFER_MEDIA_PROBE")
	if path == "" {
		t.Skip("set SNOOFER_MEDIA_PROBE to the companion DLL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	c, err := Open(abs)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	}()
	sessions, err := c.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range sessions {
		art, err := c.Art(s.ID)
		t.Logf("%+v art=%dB err=%v", s, len(art), err)
	}
}
