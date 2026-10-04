package controller

import (
	"context"
	"testing"
)

func TestVoltPlaybackControllerConverges(t *testing.T) {
	c, b := recorderController(t)
	c.Config.Studio.ASIOPlayback = true
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
