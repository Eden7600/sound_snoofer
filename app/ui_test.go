package app

import (
	"strings"
	"testing"

	"sound-snoofer/snoofer"
)

func TestGenericSectionsAndUnavailableControls(t *testing.T) {
	s := screen{width: 120, height: 30, state: ViewState{Controls: []snoofer.Control{
		{ID: "x.normal", Label: "Normal choice", Group: "Normal microphone", Subdued: true, Available: true, Status: "VR overriding — editable"},
		{ID: "x.transport", Group: "Transport", SurfaceOnly: true, Label: "Start recording"},
		{ID: "external.light", Group: "Lighting", Label: "Light", Value: "Off"},
	}}}
	if len(s.rows()) != 1 || len(s.groups()) != 2 {
		t.Fatal(s.rows())
	}
	text := s.View().Content
	for _, want := range []string{"VR overriding", "Normal choice", "Lighting"} {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(want)) {
			t.Fatalf("missing %s: %s", want, text)
		}
	}
	if strings.Contains(text, "Start recording") {
		t.Fatal("Actions returned")
	}
}
