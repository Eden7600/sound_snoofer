package app

import (
	"context"
	"fmt"
	"sort"
	"sound-snoofer/snoofer"
	"strings"
	"time"
)

// ViewState crosses the private controls pipe without domain-specific types.
type ViewState struct {
	Controls     []snoofer.Control
	Plugins      map[string]string
	PluginLabels map[string]string
	Enabled      map[string]bool
	Notice       string
	Confirmation string
}

// UIAction is either semantic input or a core lifecycle operation.
type UIAction struct {
	ID           string `json:",omitempty"`
	Request      *snoofer.Request
	Kind, Plugin string
	Enable       bool
	complete     snoofer.Completion
	session      *actionSession
}

// ActionResult belongs to one private controls-session action, never view state.
type ActionResult struct {
	ID     string
	Status snoofer.OutcomeStatus
	Detail string
}

func (a UIAction) reply(outcome snoofer.Outcome) {
	if a.complete != nil {
		a.complete(outcome)
	}
}

// stopHost uses a bounded independent cleanup deadline after run cancellation.
func stopHost(h *snoofer.Host) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return h.Stop(ctx)
}

func confirmation(before, after snoofer.Config) string {
	changes := []string{}
	for id, p := range after.Plugins {
		if before.Plugins[id].Enabled != p.Enabled {
			changes = append(changes, fmt.Sprintf("%s=%t", id, p.Enabled))
		}
	}
	sort.Strings(changes)
	return "Apply " + strings.Join(changes, ", ") + " and restart Snoofer?"
}
