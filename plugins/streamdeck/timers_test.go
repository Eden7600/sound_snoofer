package streamdeck

import (
	"testing"
	"time"

	"sound-snoofer/snoofer"
)

func TestTimerPanels(t *testing.T) {
	now := time.Unix(1000, 0)
	shown := map[string]snoofer.Control{
		"soundboard.status": {ID: "soundboard.status"},
		"hue.status":        {ID: "hue.status", Timers: []snoofer.Timer{{Control: "hue.x", Ends: now.Add(time.Hour)}}},
	}
	var timers []snoofer.Timer
	for n, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"} {
		id := "soundboard.clip-" + name
		shown[id] = snoofer.Control{ID: id, Label: name, Icon: "soundboard-play", Artwork: "art-" + name}
		timers = append(timers, snoofer.Timer{Control: id, Ends: now.Add(time.Duration(n)*time.Minute + 1500*time.Millisecond)})
	}
	page := Page{ID: "sounds"}
	page.Keys[35] = Binding{Control: "soundboard.stop"}
	page.Dials[0] = Binding{Control: "soundboard.volume"}
	use := func(n int) {
		status := shown["soundboard.status"]
		status.Timers = timers[:n]
		shown["soundboard.status"] = status
	}

	use(2) // One per panel, oldest first, from dial 2; Hue timers stay off this page.
	panels := timerPanels(page, shown, now)
	if len(panels) != 2 || panels[1][0].Label != "a" || panels[1][0].Time != "0:02" || panels[2][0].Time != "1:02" || panels[1][1].Label != "" || panels[1][0].Artwork != "art-a" {
		t.Fatalf("two timers %+v", panels)
	}
	use(6) // Two per panel.
	panels = timerPanels(page, shown, now)
	if len(panels) != 3 || panels[1][1].Label != "b" || panels[3][1].Label != "f" {
		t.Fatalf("six timers %+v", panels)
	}
	use(9) // Beyond capacity, the newest eight.
	panels = timerPanels(page, shown, now)
	if panels[1][0].Label != "b" || panels[4][1].Label != "i" {
		t.Fatalf("nine timers %+v", panels)
	}
	// Pages without the provider's controls show nothing.
	home := Page{ID: "home"}
	home.Keys[0] = Binding{Control: "audio.mic-mute"}
	if panels := timerPanels(home, shown, now); panels != nil {
		t.Fatal("timers on Home", panels)
	}
	if remaining(-time.Second) != "0:00" || remaining(61*time.Second) != "1:01" || remaining(100*time.Millisecond) != "0:01" {
		t.Fatal("remaining format")
	}
}
