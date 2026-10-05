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
	l := Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}}}
	l.Pages[0].Dials[0] = Binding{Control: "p.command"}
	if l.Validate([]snoofer.Control{{ID: "p.command", Operations: []string{"press"}}}) == nil {
		t.Fatal("command bound to dial")
	}
}

func TestRejectOutOfBoundsJSON(t *testing.T) {
	data := []byte(`{"home":"a","pages":[{"id":"a","name":"A","dials":[{},{},{},{},{},{}]}]}`)
	var l Layout
	if l.UnmarshalJSON(data) == nil {
		t.Fatal("reserved sixth dial silently discarded")
	}
}
