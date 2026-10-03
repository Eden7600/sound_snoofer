package routing

import (
	"fmt"
	"testing"
)

func TestMicOffDisconnectsInputsKeepsOutputs(t *testing.T) {
	for _, scenario := range []string{"normal", "legacy-disabled", "webcam-missing", "webcam-ambiguous", "no-playback", "volt-missing"} {
		t.Run(scenario, func(t *testing.T) {
			c, s := voiceFixture(t)
			p, err := Build(c, s)
			if err != nil {
				t.Fatal(err)
			}
			applyPlan(&s, p)
			c.Intent = c.VoiceIntent()
			c.Intent.Source = "off"
			switch scenario {
			case "legacy-disabled":
				c.Intent.Source = "desk"
				c.Intent.Enabled = false
			case "webcam-missing":
				s.Devices[2].Available = false
			case "webcam-ambiguous":
				s.Devices = append(s.Devices, s.Devices[2])
			case "no-playback":
				s.Devices[3].Available = false
			case "volt-missing":
				s.Devices[1].Available = false
			}
			s.Assignments["input:1"] = "old mic"
			s.Assignments["input:2"] = "old mic"
			s.Assignments["A3"] = "unmanaged output"
			p, err = Build(c, s)
			if err != nil {
				t.Fatal(err)
			}
			applyPlan(&s, p)
			for _, slot := range []string{"input:1", "input:2", "input:3"} {
				if s.Assignments[slot] != "" {
					t.Fatalf("%s remains assigned: %q", slot, s.Assignments[slot])
				}
			}
			for n := 0; n < 4; n++ {
				if s.Numbers[fmt.Sprintf("Patch.asio[%d]", n)] != 0 {
					t.Fatalf("ASIO patch %d remains connected", n)
				}
			}
			if s.Assignments["A3"] != "unmanaged output" {
				t.Fatal("unmanaged output changed")
			}
			if scenario != "volt-missing" {
				if s.Assignments["A1"] != "Volt ASIO" || s.Assignments["A2"] != "speakers" {
					t.Fatal("Off changed output topology", s.Assignments)
				}
				if scenario != "no-playback" && s.Numbers["Strip[5].A2"] != 1 {
					t.Fatal("Off disconnected computer playback")
				}
			}
			p, err = Build(c, s)
			if err != nil || p.HasChanges() {
				t.Fatalf("Off did not converge: %v", err)
			}
			if scenario == "normal" {
				for _, source := range []string{"desk", "lav", "webcam"} {
					c.Intent.Source = source
					c.Intent.Enabled = true
					p, err = Build(c, s)
					if err != nil {
						t.Fatal(err)
					}
					applyPlan(&s, p)
					if s.Assignments["input:3"] != "webcam" || s.Assignments["A1"] != "Volt ASIO" || s.Assignments["A2"] != "speakers" {
						t.Fatal("restore changed outputs or failed to reconnect webcam")
					}
					for n, want := range []float32{1, 1, 2, 2} {
						if s.Numbers[fmt.Sprintf("Patch.asio[%d]", n)] != want {
							t.Fatal("ASIO input not restored", n)
						}
					}
				}
			}
		})
	}
}

func TestMicOffPreservesUnknownInput(t *testing.T) {
	c, s := voiceFixture(t)
	c.Intent = c.VoiceIntent()
	c.Intent.Source = "off"
	s.Assignments["input:3"] = "unmanaged input"
	if _, err := Build(c, s); err == nil {
		t.Fatal("unmanaged input accepted")
	}
}
