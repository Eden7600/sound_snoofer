//go:build windows && amd64

package soundboard_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"sound-snoofer/plugins/audio"
	"sound-snoofer/plugins/soundboard"
	"sound-snoofer/plugins/vr"
	"sound-snoofer/snoofer"
)

// Opt-in applies the configured routing; the main application must be stopped.
// No clip or recorder transport is started by this integration check.
func TestLiveSoundboardRouting(t *testing.T) {
	path := os.Getenv("SNOOFER_SOUNDBOARD_TEST_CONFIG")
	if path == "" {
		t.Skip("set SNOOFER_SOUNDBOARD_TEST_CONFIG with Snoofer stopped")
	}
	cfg, _, err := snoofer.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for id, p := range cfg.Plugins {
		if id != "audio" && id != "soundboard" && id != "vr" {
			p.Enabled = false
			cfg.Plugins[id] = p
		}
	}
	registry := snoofer.NewControls()
	host := snoofer.New(cfg, snoofer.Services{Path: path, Live: true, Controls: registry}, audio.Plugin(), soundboard.Plugin(), vr.Plugin())
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	host.Start(ctx)
	defer func() {
		stopCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := host.Stop(stopCtx); err != nil {
			t.Error(err)
		}
	}()
	for {
		controls := registry.Snapshot()
		for _, c := range controls {
			if strings.HasPrefix(c.ID, "soundboard.clip-") && c.Available {
				t.Logf("Ready: %s; VAIO3 destinations observed", c.Label)
				return
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("soundboard not ready: host=%v controls=%+v", host.Status(), controls)
		case <-time.After(100 * time.Millisecond):
		}
	}
}
