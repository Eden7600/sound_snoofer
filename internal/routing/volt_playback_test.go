package routing

import (
	"slices"
	"sound-snoofer/internal/model"
	"testing"
)

func TestVoltPlaybackPriorityAndMigration(t *testing.T) {
	c, s := voiceFixture(t)
	c.Studio.ASIOPlayback = true
	c.Intent = c.VoiceIntent()
	s.Devices = append(s.Devices, model.Device{Name: "AirPods", Driver: "wdm", Direction: "output", Available: true})
	apply := func(want string) {
		t.Helper()
		p, err := Build(c, s)
		if err != nil {
			t.Fatal(err)
		}
		if p.Topology.PlaybackTarget != want {
			t.Fatalf("target=%s want=%s", p.Topology.PlaybackTarget, want)
		}
		a1 := 0
		for _, op := range p.Topology.Operations {
			if op.Target == "A1" {
				a1++
			}
		}
		if a1 > 1 {
			t.Fatal("duplicate A1 assignment")
		}
		applyPlan(&s, p)
		for _, bus := range []string{"A1", "A2"} {
			v := float32(0)
			if bus == want {
				v = 1
			}
			if s.Numbers["Strip[5]."+bus] != v {
				t.Fatal("playback routing", s.Numbers)
			}
		}
	}
	apply("A2")
	if s.Assignments["A2"] != "AirPods" {
		t.Fatal("priority")
	}
	s.Devices[len(s.Devices)-1].Available = false
	apply("A2")
	if s.Assignments["A2"] != "speakers" {
		t.Fatal("speaker priority")
	}
	s.Devices[3].Available = false
	apply("A1")
	if s.Assignments["A1"] != "Volt ASIO" || s.Assignments["A2"] != "" {
		t.Fatal(s.Assignments)
	}
	s.Devices[3].Available = true
	apply("A2")
	c.Intent.PlaybackDevice = "Volt ASIO"
	if err := c.Intent.Validate(c); err != nil {
		t.Fatal(err)
	}
	apply("A1")
	c.Intent.Source = "off"
	c.Intent.Enabled = false
	apply("A1")
	if s.Assignments["A1"] != "Volt ASIO" {
		t.Fatal("Off released Volt")
	}
	s.Devices[1].Available = false
	if slices.Contains(PlaybackOptions(c, s), "Volt ASIO") {
		t.Fatal("driver-only offered")
	}
	apply("A1")
	if s.Assignments["A1"] != "speakers" {
		t.Fatal("disconnect fallback")
	}
	s.Devices[1].Available = true
	apply("A1")
	if s.Assignments["A1"] != "Volt ASIO" {
		t.Fatal("preference not restored")
	}
	s.Devices = append(s.Devices, s.Devices[1])
	if slices.Contains(PlaybackOptions(c, s), "Volt ASIO") {
		t.Fatal("ambiguous Volt offered")
	}
}
