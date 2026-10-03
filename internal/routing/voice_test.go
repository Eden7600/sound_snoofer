package routing

import (
	"fmt"
	"testing"
	"voice-snooter/internal/config"
	"voice-snooter/internal/model"
)

func voiceFixture(t *testing.T) (config.Config, model.Snapshot) {
	c, s := studioFixture(t)
	s.Edition = 3
	c.Studio.Voice = &config.Voice{}
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	for _, slot := range model.Slots(3) {
		if _, ok := s.Assignments[slot]; !ok {
			s.Assignments[slot] = ""
		}
	}
	for strip := 0; strip < 8; strip++ {
		for bus := 1; bus <= 3; bus++ {
			s.Numbers[fmt.Sprintf("Strip[%d].B%d", strip, bus)] = 0
		}
	}
	return c, s
}
func applyPlan(s *model.Snapshot, p Plan) {
	ops := p.Topology.Operations
	if len(p.Topology.Transition) > 0 {
		ops = p.Topology.Transition
	}
	for _, op := range ops {
		if op.Device != nil {
			s.Assignments[op.Target] = op.Device.Name
		} else {
			s.Numbers[op.Parameter] = float32(op.Value)
		}
	}
}
func TestVoiceMatrix(t *testing.T) {
	for _, source := range []string{"desk", "lav", "webcam"} {
		for _, mode := range []string{"direct", "element"} {
			for _, monitor := range []string{"off", "pre", "post"} {
				for _, enabled := range []bool{false, true} {
					t.Run(fmt.Sprint(source, mode, monitor, enabled), func(t *testing.T) {
						c, s := voiceFixture(t)
						i := c.VoiceIntent()
						i.Source = source
						i.Mode = mode
						i.Monitor = monitor
						i.Enabled = enabled
						c.Intent = i
						// Start with an invalid loop and competing voice sends, requiring cleanup.
						s.Numbers["Strip[6].B2"] = 1
						s.Numbers["Strip[0].B3"] = 1
						s.Numbers["Strip[5].B3"] = 1
						p, e := Build(c, s)
						if e != nil {
							t.Fatal(e)
						}
						applyPlan(&s, p)
						src := map[string]int{"desk": 0, "lav": 1, "webcam": 2}[source]
						for strip := 0; strip < 8; strip++ {
							for bus := 2; bus <= 3; bus++ {
								want := float32(0)
								if enabled && ((mode == "direct" && strip == src && bus == 3) || (mode == "element" && ((strip == src && bus == 2) || (strip == 6 && bus == 3)))) {
									want = 1
								}
								param := fmt.Sprintf("Strip[%d].B%d", strip, bus)
								if s.Numbers[param] != want {
									t.Fatal(param, s.Numbers[param], want)
								}
							}
						}
						for _, strip := range []int{0, 1, 2, 6} {
							for bus := 1; bus <= 5; bus++ {
								want := float32(0)
								if enabled && bus == 2 && ((monitor == "pre" && strip == src) || (monitor == "post" && mode == "element" && strip == 6)) {
									want = 1
								}
								if s.Numbers[fmt.Sprintf("Strip[%d].A%d", strip, bus)] != want {
									t.Fatal("monitor", strip, bus)
								}
							}
						}
						p, e = Build(c, s)
						if e != nil || p.HasChanges() || len(p.Topology.Transition) != 0 {
							t.Fatal("not converged", e, p)
						}
					})
				}
			}
		}
	}
}
func TestVoiceAvailability(t *testing.T) {
	c, s := voiceFixture(t)
	c.Intent = c.VoiceIntent()
	c.Intent.Source = "lav"
	s.Devices[1].Available = false
	p, e := Build(c, s)
	if e != nil || p.Topology.Voice.Effective != "webcam" {
		t.Fatal(p, e)
	}
	applyPlan(&s, p)
	if s.Assignments["input:3"] != "webcam" || s.Assignments["input:1"] != "" {
		t.Fatal(s.Assignments)
	}
	s.Devices[1].Available = true
	p, e = Build(c, s)
	if e != nil || p.Topology.Voice.Effective != "lav" {
		t.Fatal(p, e)
	}
	s.Devices[1].Available = false
	s.Devices[2].Available = false
	p, e = Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	applyPlan(&s, p)
	if s.Numbers["Strip[6].B3"] != 0 || !p.HasUnresolved() {
		t.Fatal(p)
	}
	s.Assignments["input:3"] = "unrelated"
	if _, e = Build(c, s); e == nil {
		t.Fatal("input ownership")
	}
}

func TestVoiceAmbiguityAndMonitoringMigration(t *testing.T) {
	c, s := voiceFixture(t)
	s.Devices[1].Available = false
	s.Devices = append(s.Devices, s.Devices[2])
	p, e := Build(c, s)
	if e != nil || p.Topology.Voice.Effective != "unavailable" {
		t.Fatal("ambiguous webcam", p, e)
	}
	c, s = voiceFixture(t)
	c.Intent = c.VoiceIntent()
	c.Intent.Monitor = "post"
	p, _ = Build(c, s)
	applyPlan(&s, p)
	s.Devices[1].Available = false
	p, e = Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	applyPlan(&s, p)
	if s.Numbers["Strip[6].A1"] != 1 || s.Numbers["Strip[6].A2"] != 0 || p.Topology.Voice.Effective != "webcam" {
		t.Fatal("monitor did not follow playback")
	}
	s.Devices[1].Available = true
	p, e = Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	applyPlan(&s, p)
	if s.Numbers["Strip[6].A2"] != 1 || s.Numbers["Strip[6].A1"] != 0 {
		t.Fatal("monitor did not return")
	}
	c.Intent.Source = "webcam"
	p, e = Build(c, s)
	if e != nil || p.Topology.Voice.Effective != "webcam" {
		t.Fatal("explicit webcam changed")
	}
}
func TestVoiceNoPlaybackAndDisabledApp(t *testing.T) {
	c, s := voiceFixture(t)
	c.Intent = c.VoiceIntent()
	c.Intent.Playback["virtual:1"] = false
	c.Intent.Monitor = "post"
	p, e := Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	applyPlan(&s, p)
	if s.Numbers["Strip[5].A2"] != 0 {
		t.Fatal("disabled app")
	}
	s.Devices[3].Available = false
	p, e = Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	applyPlan(&s, p)
	if s.Numbers["Strip[0].B2"] != 1 || s.Numbers["Strip[6].B3"] != 1 || s.Numbers["Strip[6].A2"] != 0 {
		t.Fatal(s.Numbers)
	}
}
