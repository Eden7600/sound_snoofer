package streamdeck

import (
	"testing"

	"sound-snoofer/snoofer"
)

func TestPages(t *testing.T) {
	l := Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}}
	if l.next("a", -1) != "b" || l.next("b", 1) != "a" {
		t.Fatal("wrap")
	}
	l.SharedKeys[0] = Binding{Control: "missing.control", Label: "Saved"}
	if err := l.Validate(nil); err != nil {
		t.Fatal(err)
	}
	if l.effective("b").Keys[0].Label != "Saved" {
		t.Fatal("shared")
	}
	draft := l.clone()
	draft.Pages[0].Keys[0] = Binding{Control: "a.x"}
	if draft.Validate(nil) == nil {
		t.Fatal("collision")
	}
	if l.Pages[0].Keys[0].Control != "" {
		t.Fatal("draft mutated original")
	}
	if err := l.remove("a"); err != nil || l.Home != "b" {
		t.Fatal(l, err)
	}
	if l.remove("b") == nil {
		t.Fatal("deleted final page")
	}
}
func TestBindingCompatibility(t *testing.T) {
	for _, tt := range []struct {
		name      string
		control   snoofer.Control
		key, dial bool
	}{
		{"status", snoofer.Control{Kind: "status"}, true, false},
		{"status with operations", snoofer.Control{Kind: "status", Operations: []string{"press", "adjust"}}, true, false},
		{"hidden status", snoofer.Control{Kind: "status", Hidden: true}, true, false},
		{"text", snoofer.Control{Kind: "text", Operations: []string{"press", "set", "adjust"}}, false, false},
		{"command", snoofer.Control{Kind: "command", Operations: []string{"press"}}, true, false},
		{"selection", snoofer.Control{Kind: "selection", Operations: []string{"set"}}, true, false},
		{"gain", snoofer.Control{Kind: "numeric", Operations: []string{"adjust"}}, false, true},
		{"gain with mute", snoofer.Control{Kind: "numeric", Operations: []string{"press", "adjust"}}, true, true},
		{"connection", snoofer.Control{Kind: "connection"}, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.control.ID = "p.control"
			for _, dial := range []bool{false, true} {
				l := Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}}}
				want := tt.key
				if dial {
					l.Pages[0].Dials[0] = Binding{Control: tt.control.ID}
					want = tt.dial
				} else {
					l.Pages[0].Keys[0] = Binding{Control: tt.control.ID}
				}
				if err := l.Validate([]snoofer.Control{tt.control}); (err == nil) != want {
					t.Fatalf("dial=%v: valid=%v, want %v: %v", dial, err == nil, want, err)
				}
			}
		})
	}
}

func TestRejectOutOfBoundsJSON(t *testing.T) {
	data := []byte(`{"home":"a","pages":[{"id":"a","name":"A","dials":[{},{},{},{},{},{}]}]}`)
	var l Layout
	if l.UnmarshalJSON(data) == nil {
		t.Fatal("reserved sixth dial silently discarded")
	}
}
