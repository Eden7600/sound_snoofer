//go:build windows

package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func TestCoreDesktopSmoke(t *testing.T) {
	if os.Getenv("SNOOFER_DESKTOP_SMOKE") != "1" {
		t.Skip("requires interactive Windows desktop")
	}
	path := filepath.Join(t.TempDir(), "snoofer.json")
	if _, err := snoofer.Save(path, snoofer.Config{Version: 1, Plugins: map[string]snoofer.PluginConfig{}}, "missing"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := Run(ctx, []string{"--dry-run", "--config", path}); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() == nil {
		t.Fatal("did not own isolated tray lifecycle")
	}
}
