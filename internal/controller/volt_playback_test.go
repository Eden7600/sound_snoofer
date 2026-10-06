package controller

import (
	"context"
	"sound-snoofer/internal/config"
	"testing"
)

func TestVoltPlaybackControllerConverges(t *testing.T) {
	c, b := recorderController(t)
	c.Config.Studio.Playback = append(c.Config.Studio.Playback, config.Candidate{Driver: "asio", Pattern: c.Config.Studio.ASIO[0].ASIOPattern, Regex: c.Config.Studio.ASIO[0].ASIORegex})
	c.Config.Intent.PlaybackDevice = "Volt ASIO"
	for _, choice := range []string{"Volt ASIO", "", "Volt ASIO"} {
		c.Config.Intent.PlaybackDevice = choice
		p, err := c.Plan()
		if err != nil {
			t.Fatal(err)
		}
		if err = c.Apply(context.Background(), p); err != nil {
			t.Fatal(err)
		}
		want := "A1"
		if choice == "" {
			want = "A2"
		}
		p, err = c.Plan()
		if err != nil || p.HasChanges() || p.Topology.PlaybackTarget != want {
			t.Fatal("not converged", err, p)
		}
		if b.s.Assignments["A1"] != "Volt ASIO" {
			t.Fatal("ASIO released")
		}
	}
}
