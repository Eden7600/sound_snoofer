package streamdeck

import (
	"fmt"
	"testing"

	"sound-snoofer/snoofer"
)

func TestAutomaticPagesPreserveBindingsAndOverflow(t *testing.T) {
	l := Layout{Home: "home", Pages: []Page{{ID: "home", Name: "Home"}, {ID: "sounds", Name: "Soundboard", AutoControls: "soundboard.clip-"}}}
	l.Pages[0].Keys[31] = Binding{Control: "core.open-controls"}
	l.Pages[1].Keys[35] = Binding{Control: "soundboard.stop"}
	l.SharedKeys[34] = Binding{Control: "shared"}
	controls := []snoofer.Control{}
	for n := 0; n < 70; n++ {
		controls = append(controls, snoofer.Control{ID: fmt.Sprintf("soundboard.clip-%02d", n), Label: fmt.Sprintf("Clip %02d", n), Operations: []string{"press"}})
	}
	expanded := l.expanded(controls)
	if len(expanded.Pages) != 4 || expanded.Pages[0] != l.Pages[0] {
		t.Fatal("overflow or Home changed")
	}
	for _, p := range expanded.Pages[1:] {
		if p.Keys[35].Control != "soundboard.stop" || p.Keys[34].Control != "" {
			t.Fatal("manual/shared binding replaced")
		}
	}
	if expanded.Pages[2].Keys[0].Control != "soundboard.clip-34" || expanded.next("sounds", 1) != "sounds~auto~2" {
		t.Fatal("pagination broken")
	}
	if got := expanded.pageNames("sounds"); got != [3]string{"Home", "Soundboard", "Soundboard 2"} {
		t.Fatal("automatic page neighbors", got)
	}
	if l.Pages[1].Keys[0].Control != "" {
		t.Fatal("saved layout mutated")
	}
	if len(l.expanded(nil).Pages) != 2 {
		t.Fatal("empty catalogue left overflow pages")
	}
}
