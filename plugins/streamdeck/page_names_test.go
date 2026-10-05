package streamdeck

import "testing"

func TestPageNamesFollowCyclicOrder(t *testing.T) {
	l := Layout{Home: "a", Pages: []Page{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}}}
	for id, want := range map[string][3]string{"a": {"C", "A", "B"}, "b": {"A", "B", "C"}, "c": {"B", "C", "A"}, "missing": {"C", "A", "B"}} {
		if got := l.pageNames(id); got != want {
			t.Fatalf("%s: %v, want %v", id, got, want)
		}
	}
	l.Pages = l.Pages[:2]
	if got := l.pageNames("a"); got != [3]string{"B", "A", "B"} {
		t.Fatal(got)
	}
	l.Pages = l.Pages[:1]
	if got := l.pageNames("a"); got != [3]string{"A", "A", "A"} {
		t.Fatal(got)
	}
}
