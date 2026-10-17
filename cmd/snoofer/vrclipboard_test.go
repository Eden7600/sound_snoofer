//go:build !core && !no_vr && !no_vrclipboard

package main

import (
	"testing"

	"sound-snoofer/snoofer"
)

func TestVRClipboardIsIndependentAndDisabledByDefault(t *testing.T) {
	config := snoofer.Defaults(plugins)
	clipboard, ok := config.Plugins["vrclipboard"]
	if !ok || clipboard.Enabled {
		t.Fatal("VR Clipboard missing or enabled automatically")
	}
	for id, plugin := range config.Plugins {
		plugin.Enabled = id == "vrclipboard"
		config.Plugins[id] = plugin
	}
	if err := snoofer.ValidateEnabled(config, plugins...); err != nil {
		t.Fatal(err)
	}
}
