package app

import (
	"errors"
	"testing"
)

func TestStartupCommandQuotesPaths(t *testing.T) {
	got := startupCommand(`C:\Program Files\Snoofer\snoofer.exe`, `C:\Program Files\Snoofer\snoofer.json`)
	want := `"C:\Program Files\Snoofer\snoofer.exe" --config "C:\Program Files\Snoofer\snoofer.json"`
	if got != want {
		t.Fatalf("command = %s, want %s", got, want)
	}
}

func TestStartupControlShowsObservedRegistration(t *testing.T) {
	const command = `"C:\bin\snoofer.exe" --config "C:\bin\snoofer.json"`
	for _, tc := range []struct {
		name      string
		observed  startupState
		readErr   error
		live      bool
		value     string
		status    string
		available bool
	}{
		{name: "absent", live: true, value: "Off", available: true},
		{name: "this copy", observed: startupState{Command: command}, live: true, value: "On", available: true},
		{name: "case differs", observed: startupState{Command: `"c:\BIN\snoofer.exe" --config "c:\bin\SNOOFER.json"`}, live: true, value: "On", available: true},
		{name: "disabled in Windows", observed: startupState{Command: command, Disabled: true}, live: true, value: "Off", status: "Disabled in Windows startup apps", available: true},
		{name: "another copy", observed: startupState{Command: `"D:\old\snoofer.exe"`}, live: true, value: "Off", status: `Set for another copy: "D:\old\snoofer.exe"`, available: true},
		{name: "read failure", readErr: errors.New("read login startup: access denied"), live: true, value: "N/A", status: "read login startup: access denied"},
		{name: "preview", observed: startupState{Command: command}, value: "On", status: "Preview"},
		{name: "preview with note", observed: startupState{Command: command, Disabled: true}, value: "Off", status: "Preview · Disabled in Windows startup apps"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := startupControl(tc.observed, tc.readErr, command, tc.live)
			if c.ID != "core.startup" || c.Kind != "toggle" || len(c.Operations) != 1 || c.Operations[0] != "press" {
				t.Fatalf("control shape = %+v", c)
			}
			if c.Value != tc.value || c.Status != tc.status || c.Available != tc.available {
				t.Fatalf("value/status/available = %q/%q/%t, want %q/%q/%t", c.Value, c.Status, c.Available, tc.value, tc.status, tc.available)
			}
		})
	}
}
