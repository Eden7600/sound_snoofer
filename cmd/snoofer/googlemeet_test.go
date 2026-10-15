//go:build !core && !no_googlemeet

package main

import (
	"testing"

	"sound-snoofer/snoofer"
)

func TestGoogleMeetRegistrationIsIndependentAndDisabledByDefault(t *testing.T) {
	config := snoofer.Defaults(plugins)
	meet, ok := config.Plugins["googlemeet"]
	if !ok || meet.Enabled {
		t.Fatal("Meet registration missing or enabled automatically")
	}
	for id, plugin := range config.Plugins {
		plugin.Enabled = id == "googlemeet"
		config.Plugins[id] = plugin
	}
	if err := snoofer.ValidateEnabled(config, plugins...); err != nil {
		t.Fatal("Meet requires another plugin", err)
	}
}
