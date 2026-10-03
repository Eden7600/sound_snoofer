package routing

import (
	"fmt"
	"testing"
)

func TestMasterMicOffAllSends(t *testing.T) {
	for _, state := range []string{"normal", "no-profile", "unknown", "active-conflict", "disconnected", "ambiguous", "source-off"} {
		t.Run(state, func(t *testing.T) {
			c, s := recordingFixture(t)
			c.Intent = c.VoiceIntent()
			c.Intent.Recording.MicEnabled = true
			c.Intent.Recording.ComputerEnabled = true
			c.Intent.Monitor = "post"
			c.Intent.Enabled = false
			switch state {
			case "source-off":
				c.Intent.Source = "off"
				c.Intent.Enabled = true
			case "no-profile":
				c.Studio.Recording = nil
				c.Intent.Recording = nil
			case "unknown":
				s.Recorder = nil
			case "active-conflict":
				s.Recorder.Values["Recorder.record"] = 1
				s.Recorder.Values["Recorder.stop"] = 0
			case "disconnected":
				s.Devices[1].Available = false
			case "ambiguous":
				s.Devices[1].Available = false
				s.Devices = append(s.Devices, s.Devices[2])
			}
			for _, n := range []int{0, 1, 2, 6} {
				for _, letter := range []string{"A", "B"} {
					count := 5
					if letter == "B" {
						count = 3
					}
					for bus := 1; bus <= count; bus++ {
						s.Numbers[fmt.Sprintf("Strip[%d].%s%d", n, letter, bus)] = 1
					}
				}
			}
			s.Numbers["Strip[5].B1"] = 1
			p, e := Build(c, s)
			if e != nil {
				t.Fatal(e)
			}
			applyPlan(&s, p)
			for _, n := range []int{0, 1, 2, 6} {
				for _, letter := range []string{"A", "B"} {
					count := 5
					if letter == "B" {
						count = 3
					}
					for bus := 1; bus <= count; bus++ {
						param := fmt.Sprintf("Strip[%d].%s%d", n, letter, bus)
						if s.Numbers[param] != 0 {
							t.Fatal(param)
						}
					}
				}
			}
			if s.Numbers["Strip[5].B1"] != 1 {
				t.Fatal("computer capture changed")
			}
			if (state != "source-off" && c.Intent.Source != "desk") || c.Intent.Mode != "element" || c.Intent.Monitor != "post" {
				t.Fatal("preferences lost")
			}
			if state == "normal" {
				c.Intent.Enabled = true
				p, e = Build(c, s)
				if e != nil {
					t.Fatal(e)
				}
				applyPlan(&s, p)
				if s.Numbers["Strip[0].B2"] != 1 || s.Numbers["Strip[6].B3"] != 1 || s.Numbers["Strip[6].A2"] != 1 || s.Numbers["Strip[0].B1"] != 1 {
					t.Fatal("On did not restore")
				}
			}
		})
	}
}
