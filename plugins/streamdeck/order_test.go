package streamdeck

import (
	"testing"

	"sound-snoofer/snoofer"
)

func TestRegionFillsByCollectionOrder(t *testing.T) {
	page := Page{ID: "apps", Name: "Apps", Regions: []Region{{Source: "appaudio.apps", First: 0, Last: 2}}}
	l := Layout{Home: "apps", Pages: []Page{page}}
	controls := []snoofer.Control{
		{ID: "appaudio.app-a", Label: "Alpha", Collection: "appaudio.apps", Order: 3, Operations: []string{"press"}},
		{ID: "appaudio.app-b", Label: "Beta", Collection: "appaudio.apps", Order: 1, Operations: []string{"press"}},
		{ID: "appaudio.app-c", Label: "Gamma", Collection: "appaudio.apps", Order: 2, Operations: []string{"press"}},
	}
	keys := l.expanded(controls).Pages[0].Keys
	if keys[0].Control != "appaudio.app-b" || keys[1].Control != "appaudio.app-c" || keys[2].Control != "appaudio.app-a" {
		t.Fatal("fill order", keys[:3])
	}
}
