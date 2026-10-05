package streamdeck

import (
	"sound-snoofer/snoofer"
	"testing"
)

func TestEditorPreviewOwnership(t *testing.T) {
	l := DefaultLayout()
	p := l.Pages[0].ID
	l.Pages[0].Keys = [Keys]Binding{}
	l.Pages[0].AutoControls = "clips."
	l.SharedKeys[3] = Binding{Control: "shared.x", Label: "Shared"}
	l.Pages[0].Keys[7] = Binding{Control: "manual.x", Label: "Manual"}
	view := editorView(l, p, 7, true, []snoofer.Control{{ID: "clips.a", Label: "Clip", Operations: []string{"press"}}})
	if len(view.Keys) != 36 || len(view.Dials) != 5 || view.Selected != 7 || !view.Dirty {
		t.Fatal(view)
	}
	if view.Keys[0].Source != "Auto" || view.Keys[3].Source != "Shared" || view.Keys[7].Source != "" {
		t.Fatal(view.Keys)
	}
	if l.Pages[0].Keys[0].Control != "" {
		t.Fatal("preview mutated draft")
	}
}
