package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sound-snoofer/snoofer"
)

func testLifecycle(t *testing.T) (*lifecycle, UIAction) {
	t.Helper()
	config := snoofer.Config{Version: 1, Plugins: map[string]snoofer.PluginConfig{
		"audio": {Settings: json.RawMessage(`{"gain":1}`)}, "deck": {Settings: json.RawMessage(`{}`)}, "extra": {Settings: json.RawMessage(`{}`)},
	}}
	path := filepath.Join(t.TempDir(), "settings.json")
	revision, err := snoofer.Save(path, config, "missing")
	if err != nil {
		t.Fatal(err)
	}
	host := snoofer.New(config, snoofer.Services{}, snoofer.Plugin{ID: "audio"}, snoofer.Plugin{ID: "deck", Requires: []string{"audio"}}, snoofer.Plugin{ID: "extra"})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	session := newActionSession(ctx, cancel)
	return &lifecycle{path: path, host: host}, UIAction{ID: "select-1", Kind: "selection", Plugin: "deck", Enable: true, ConfigRevision: revision, session: session}
}

func propose(t *testing.T, l *lifecycle, action UIAction) *enablementProposal {
	t.Helper()
	result, restart := l.handle(action)
	if result.Status != snoofer.Processed || restart || l.pending() == nil {
		t.Fatalf("proposal failed: %+v, restart %v", result, restart)
	}
	return l.pending()
}

func TestLifecycleProposalIsOwnedAndConsumedOnce(t *testing.T) {
	l, action := testLifecycle(t)
	p := propose(t, l, action)
	if p.source != action.ID || !strings.Contains(p.text, "audio=true") || !strings.Contains(p.text, "deck=true") {
		t.Fatal(p)
	}
	for _, invalid := range []UIAction{
		action, {Kind: "retry", session: action.session}, {Kind: "confirm", ProposalID: p.id},
		{Kind: "cancel", ProposalID: "old", session: action.session}, {Kind: "confirm", session: action.session},
	} {
		result, restart := l.handle(invalid)
		if result.Status != snoofer.Rejected || restart || l.pending() != p {
			t.Fatalf("invalid action changed proposal: %+v, %+v", invalid, result)
		}
	}
	confirm := UIAction{ID: "confirm-1", Kind: "confirm", ProposalID: p.id, session: action.session}
	result, restart := l.handle(confirm)
	if result.Status != snoofer.Processed || !restart || l.pending() != nil {
		t.Fatal(result, restart)
	}
	saved, _, err := snoofer.Load(l.path)
	if err != nil || !saved.Plugins["audio"].Enabled || !saved.Plugins["deck"].Enabled {
		t.Fatal(saved, err)
	}
	if result, restart := l.handle(confirm); result.Status != snoofer.Rejected || restart {
		t.Fatal("duplicate confirmed", result)
	}
}

func TestLifecycleStaleSelectionAndConfirmPreserveExternalConfig(t *testing.T) {
	for _, timing := range []string{"selection", "confirmation"} {
		t.Run(timing, func(t *testing.T) {
			l, action := testLifecycle(t)
			if timing == "confirmation" {
				p := propose(t, l, action)
				action.Kind, action.ProposalID = "confirm", p.id
			}
			current, revision, err := snoofer.Load(l.path)
			if err != nil {
				t.Fatal(err)
			}
			plugin := current.Plugins["audio"]
			plugin.Settings = json.RawMessage(`{"gain":99}`)
			current.Plugins["audio"] = plugin
			if _, err := snoofer.Save(l.path, current, revision); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(l.path)
			result, restart := l.handle(action)
			if (result.Status != snoofer.Rejected && result.Status != snoofer.Failed) || restart || l.pending() != nil {
				t.Fatal(result, restart)
			}
			after, _ := os.ReadFile(l.path)
			if string(after) != string(before) {
				t.Fatal("external configuration overwritten")
			}
		})
	}
}

func TestLifecycleSelectionUsesDisplayedConfigAsBase(t *testing.T) {
	l, action := testLifecycle(t)
	current, revision, err := snoofer.Load(l.path)
	if err != nil {
		t.Fatal(err)
	}
	current.Plugins["extra"] = snoofer.PluginConfig{Enabled: true, Settings: json.RawMessage(`{"preserved":true}`)}
	action.ConfigRevision, err = snoofer.Save(l.path, current, revision)
	if err != nil {
		t.Fatal(err)
	}
	p := propose(t, l, action)
	extra := p.config.Plugins["extra"]
	var value struct{ Preserved bool }
	if err := json.Unmarshal(extra.Settings, &value); err != nil || !extra.Enabled || !value.Preserved {
		t.Fatalf("proposal rebased onto running config: %+v, %v", extra, err)
	}
	if strings.Contains(p.text, "extra=") {
		t.Fatalf("unrequested change: %s", p.text)
	}
}

func TestLifecycleOldCancelCannotCancelNewProposal(t *testing.T) {
	l, action := testLifecycle(t)
	first := propose(t, l, action)
	cancel := UIAction{Kind: "cancel", ProposalID: first.id, session: action.session}
	if result, restart := l.handle(cancel); result.Status != snoofer.Processed || restart {
		t.Fatal(result)
	}
	action.ID = "select-2"
	second := propose(t, l, action)
	if second.id == first.id {
		t.Fatal("proposal ID reused")
	}
	if result, _ := l.handle(cancel); result.Status != snoofer.Rejected || l.pending() != second {
		t.Fatal(result)
	}
}

func TestLifecycleOwnerCloseAbandonsProposal(t *testing.T) {
	l, action := testLifecycle(t)
	first := propose(t, l, action)
	action.session.cancel()
	if l.pending() != nil {
		t.Fatal("closed owner retained proposal")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	action.session, action.ID = newActionSession(ctx, cancel), "new-session"
	second := propose(t, l, action)
	if second.id == first.id {
		t.Fatal("reopened session reused proposal")
	}
	if result, _ := l.handle(UIAction{Kind: "confirm", ProposalID: first.id, session: first.owner}); result.Status != snoofer.Rejected || l.pending() != second {
		t.Fatal(result)
	}
}
