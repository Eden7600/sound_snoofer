package desktop

import (
	"fmt"
	"strings"

	"sound-snoofer/internal/tui"
)

// IsTrayLaunch identifies the passive launch mode without changing explicit CLI commands.
func IsTrayLaunch(args []string) bool {
	if len(args) == 0 {
		return true
	}
	if args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return false
	}
	return args[0] == "tray" || strings.HasPrefix(args[0], "-")
}

func statusText(s tui.State) string {
	mode := "Live"
	if !s.Live {
		mode = "Preview"
	}
	if s.NeedsAttention() {
		return mode + " · Attention — open controls"
	}
	if !s.Connected {
		return mode + " · Connecting"
	}
	if s.Plan != nil && s.Plan.Topology != nil && len(s.Plan.Topology.Unresolved) > 0 {
		return mode + " · Routing unresolved"
	}
	return fmt.Sprintf("%s · Connected", mode)
}
