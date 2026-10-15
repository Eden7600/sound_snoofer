package routing

import (
	"reflect"
	"strings"
	"testing"

	"sound-snoofer/internal/model"
)

func TestHardwareTransitionGatesOnlyDependencies(t *testing.T) {
	for _, scenario := range []string{"desk-patch", "lav-patch", "webcam", "webcam-active", "playback", "asio-output"} {
		t.Run(scenario, func(t *testing.T) {
			c, s := recordingFixture(t)
			c.Intent = c.VoiceIntent()
			c.Intent.Monitor = "pre"
			c.Intent.Recording.MicEnabled = true
			c.Intent.Recording.ComputerEnabled = true
			if scenario == "webcam-active" {
				c.Intent.Source = "webcam"
			}
			p, err := Build(c, s)
			if err != nil {
				t.Fatal(err)
			}
			applyPlan(&s, p)
			want := map[string][]int{}
			switch scenario {
			case "desk-patch", "asio-output":
				want = map[string][]int{"Strip[0].B2": {0, 1}, "Strip[0].B1": {0, 1}, "Strip[0].A2": {0, 1}, "Strip[6].B3": {0, 1}}
				if scenario == "desk-patch" {
					s.Numbers["Patch.asio[0]"] = 0
				} else {
					s.Assignments["A1"] = ""
				}
			case "lav-patch":
				s.Numbers["Patch.asio[2]"] = 0
			case "webcam", "webcam-active":
				s.Devices[2].Name = "webcam replacement"
				if scenario == "webcam-active" {
					want = map[string][]int{"Strip[2].B2": {0, 1}, "Strip[2].B1": {0, 1}, "Strip[2].A2": {0, 1}, "Strip[6].B3": {0, 1}}
				}
			case "playback":
				s.Devices = append(s.Devices, model.Device{Name: "AirPods", Direction: "output", Driver: "wdm", Available: true})
				want = map[string][]int{"Strip[0].A2": {0, 1}, "Strip[3].A2": {0, 1}, "Strip[5].A2": {0, 1}}
			}
			p, err = Build(c, s)
			if err != nil {
				t.Fatal(err)
			}
			got := map[string][]int{}
			for _, op := range p.Topology.Transition {
				if !op.Change {
					continue
				}
				if strings.HasPrefix(op.Parameter, "Strip[") {
					got[op.Parameter] = append(got[op.Parameter], op.Value)
				}
				if op.Device != nil || strings.HasPrefix(op.Parameter, "Patch.") {
					for param := range want {
						if s.Numbers[param] != 0 {
							t.Fatalf("%s not gated before hardware write %+v", param, op)
						}
					}
				}
				if op.Device != nil {
					s.Assignments[op.Target] = op.Device.Name
				} else {
					s.Numbers[op.Parameter] = float32(op.Value)
				}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("send writes = %v, want %v", got, want)
			}
			p, err = Build(c, s)
			if err != nil || p.HasChanges() {
				t.Fatal("hardware transition did not converge", err)
			}
		})
	}
}

func TestTransitionGatesInterfaceMicsBeyondTheFirstTwoStrips(t *testing.T) {
	c, s := voiceFixture(t)
	for _, scenario := range []string{"patch", "a1"} {
		t.Run(scenario, func(t *testing.T) {
			// A third interface microphone on strip 2 owns Patch.asio[4] and [5].
			patch := Operation{Parameter: "Patch.asio[4]", Value: 3, BeforeValue: 3}
			topology := &Topology{ASIOActive: true}
			if scenario == "patch" {
				patch.BeforeValue, patch.Change = 0, true
			} else {
				topology.Operations = append(topology.Operations, Operation{Target: "A1", Device: &model.Device{Name: "Other ASIO", Driver: "asio"}, BeforeName: "Volt ASIO", Change: true})
			}
			send := Operation{Parameter: "Strip[2].B3", Value: 1, BeforeValue: 1}
			topology.Operations = append(topology.Operations, patch, send)
			buildVoiceTransition(c, s, topology)
			if len(topology.Transition) == 0 || topology.Transition[0].Parameter != "Strip[2].B3" || topology.Transition[0].Value != 0 {
				t.Fatalf("strip 2 not gated first: %+v", topology.Transition)
			}
		})
	}
}
