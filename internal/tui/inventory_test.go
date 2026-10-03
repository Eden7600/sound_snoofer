package tui

import (
	"strings"
	"testing"

	"voice-snooter/internal/model"
)

func TestDeviceInventoryOmitsVirtual(t *testing.T) {
	s := screen{tab: 2, width: 100, height: 30, state: State{Connected: true, Snapshot: model.Snapshot{Devices: []model.Device{{Name: "Voicemeeter Virtual ASIO"}, {Name: "Volt", Driver: "asio"}}}}}
	view := s.View().Content
	if strings.Contains(view, "Virtual ASIO") || !strings.Contains(view, "Volt") {
		t.Fatal(view)
	}
	if len(s.state.Snapshot.Devices) != 2 {
		t.Fatal("snapshot mutated")
	}
	s.state.Snapshot.Devices = s.state.Snapshot.Devices[:1]
	if !strings.Contains(s.View().Content, "No physical devices found.") {
		t.Fatal("missing empty state")
	}
}
