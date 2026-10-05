//go:build windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sound-snoofer/app"
	"sound-snoofer/snoofer"
)

func TestFullPreviewLifecycle(t *testing.T) {
	source := os.Getenv("SNOOFER_FULL_PREVIEW_CONFIG")
	if source == "" {
		t.Skip("opt-in real device read-only preview")
	}
	cfg, _, err := snoofer.Load(source)
	if err != nil {
		t.Fatal(err)
	}
	for id, p := range cfg.Plugins {
		t.Logf("%s enabled=%v", id, p.Enabled)
	}
	registry := snoofer.NewControls()
	host := snoofer.New(cfg, snoofer.Services{Controls: registry, Path: source, Live: false}, plugins...)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	host.Start(ctx)
	time.Sleep(2 * time.Second)
	for id, status := range host.Status() {
		t.Log(id, status)
		if status != "Running" && status != "Disabled" {
			t.Error(id, status)
		}
	}
	count := 0
	for _, c := range registry.Snapshot() {
		if c.ID == "audio.health" {
			t.Log("audio health:", c.Value, c.Status)
		}
		count++
	}
	if count == 0 {
		t.Error("no controls")
	}
	cancel()
	stopCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := host.Stop(stopCtx); err != nil {
		t.Fatal(err)
	}
}
func TestPluginMatrixValidation(t *testing.T) {
	for _, ids := range [][]string{{}, {"audio"}, {"media", "streamdeck"}, {"audio", "vr", "media", "streamdeck"}} {
		cfg := snoofer.Defaults(plugins)
		for _, id := range ids {
			p, compiled := cfg.Plugins[id]
			if !compiled {
				continue
			}
			p.Enabled = true
			cfg.Plugins[id] = p
		}
		if err := snoofer.ValidateEnabled(cfg, plugins...); err != nil {
			t.Fatal(ids, err)
		}
	}
	cfg := snoofer.Defaults(plugins)
	p := cfg.Plugins["vr"]
	p.Enabled = true
	cfg.Plugins["vr"] = p
	if snoofer.ValidateEnabled(cfg, plugins...) == nil {
		t.Fatal("VR without audio accepted")
	}
}
func TestNoPluginControlsProcess(t *testing.T) {
	if os.Getenv("SNOOFER_CORE_TRAY") != "1" {
		t.Skip("opt-in desktop smoke")
	}
	path := filepath.Join(t.TempDir(), "snoofer.json")
	if _, err := snoofer.Save(path, snoofer.Config{Version: 1, Plugins: map[string]snoofer.PluginConfig{}}, "missing"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := app.Run(ctx, []string{"--dry-run", "--config", path}, plugins...); err != nil {
		t.Fatal(err)
	}
}
