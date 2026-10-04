//go:build windows

package desktop

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestInstanceSignalAndRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	first, exists, err := instance(path, false)
	if err != nil || exists {
		t.Fatalf("first: %v %v", exists, err)
	}
	defer windows.CloseHandle(first)
	second, exists, err := instance(path, false)
	if err != nil || !exists {
		t.Fatalf("second: %v %v", exists, err)
	}
	windows.CloseHandle(second)
	signal, err := windows.WaitForSingleObject(first, 0)
	if err != nil || signal != windows.WAIT_OBJECT_0 {
		t.Fatalf("signal: %v %v", signal, err)
	}
	signal, err = windows.WaitForSingleObject(first, 0)
	if err != nil || signal != uint32(windows.WAIT_TIMEOUT) {
		t.Fatalf("signal not consumed: %v %v", signal, err)
	}
	other, exists, err := instance(path, true)
	if err != nil || exists {
		t.Fatalf("preview/live collision: %v %v", exists, err)
	}
	windows.CloseHandle(other)
	fresh, exists, err := instance(path, true)
	if err != nil || exists {
		t.Fatalf("lease not released: %v %v", exists, err)
	}
	windows.CloseHandle(fresh)
}

// TestTrayDesktopSmoke is an explicit read-only integration check. Normal test
// runs never open the installed Remote API or a Windows tray icon.
func TestTrayDesktopSmoke(t *testing.T) {
	path := os.Getenv("SOUND_SNOOFER_TRAY_SMOKE_CONFIG")
	if path == "" {
		t.Skip("requires an isolated profile and interactive desktop")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := RunTray(ctx, []string{"--dry-run", "--config", path}); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() == nil {
		t.Fatal("smoke profile already has a tray instance; use a distinct isolated profile")
	}
}
