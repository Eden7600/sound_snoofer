package app

import (
	"strings"

	"sound-snoofer/snoofer"
)

// startupState is the observed Windows sign-in registration.
type startupState struct {
	Command  string // The registered command line; empty when nothing is registered.
	Disabled bool   // Windows' startup-apps setting has disabled the registration.
}

// startupCommand launches the tray for one configuration. The configuration
// path must be absolute: Windows chooses the working directory at sign-in.
func startupCommand(exe, config string) string {
	return `"` + exe + `" --config "` + config + `"`
}

// startupControl presents the observed registration for this copy's command.
// Only a registration of exactly this command that Windows has not disabled
// is On. Preview publishes it unavailable so it can never be changed.
func startupControl(observed startupState, readErr error, command string, live bool) snoofer.Control {
	control := snoofer.Control{
		ID:         "core.startup",
		Label:      "Start at Login",
		Group:      "System",
		Kind:       "toggle",
		Value:      "Off",
		Available:  live,
		Operations: []string{"press"},
	}
	switch {
	case readErr != nil:
		control.Value = "N/A"
		control.Available = false
		control.Status = readErr.Error()
	case observed.Command == "":
	case !strings.EqualFold(observed.Command, command):
		control.Status = "Set for another copy: " + observed.Command
	case observed.Disabled:
		control.Status = "Disabled in Windows startup apps"
	default:
		control.Value = "On"
	}
	if !live && control.Status == "" {
		control.Status = "Preview"
	} else if !live {
		control.Status = "Preview · " + control.Status
	}
	return control
}
