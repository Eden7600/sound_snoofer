package routing

import (
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

// pausedFixture is a converged recording setup, then disturbed by scenario.
func pausedFixture(t *testing.T, disturb func(*model.Snapshot)) (config.Config, model.Snapshot) {
	t.Helper()
	c, s := recordingFixture(t)
	c.Intent = c.VoiceIntent()
	c.Intent.Monitor = "pre"
	c.Intent.Recording.MicEnabled = true
	p, err := Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	applyPlan(&s, p)
	disturb(&s)
	return c, s
}

func changed(p Plan) []Operation {
	var out []Operation
	for _, op := range append(append([]Operation(nil), p.Topology.Operations...), p.Topology.Transition...) {
		if op.Change {
			out = append(out, op)
		}
	}
	return out
}

func TestPauseDevicesHoldsAssignmentsWithoutGating(t *testing.T) {
	c, s := pausedFixture(t, func(s *model.Snapshot) { s.Assignments["A1"] = "" })
	normal, err := Build(c, s)
	if err != nil || !normal.HasChanges() || len(normal.Topology.Transition) == 0 {
		t.Fatal("fixture should need a gated device change", err)
	}
	c.Intent.PauseDevices = true
	p, err := Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if p.HasChanges() || len(changed(p)) != 0 || p.Topology.HeldDevices != 1 || p.Topology.HeldSends != 0 {
		t.Fatalf("held %d/%d, changed %+v", p.Topology.HeldDevices, p.Topology.HeldSends, changed(p))
	}
	if p.Key() != normal.Key() {
		t.Fatal("holding changed the plan key")
	}
}

func TestPauseSendsHoldsEveryParameter(t *testing.T) {
	c, s := pausedFixture(t, func(s *model.Snapshot) {
		s.Numbers["Patch.asio[0]"] = 0
		s.Numbers["Strip[0].B3"] = 1
	})
	c.Intent.PauseSends = true
	p, err := Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if p.HasChanges() || len(changed(p)) != 0 || p.Topology.HeldSends < 2 || len(p.Topology.Transition) != 0 {
		t.Fatalf("held %d, changed %+v", p.Topology.HeldSends, changed(p))
	}
	// Device changes still apply directly, without a transition.
	s.Assignments["A1"] = ""
	p, err = Build(c, s)
	if err != nil {
		t.Fatal(err)
	}
	ops := changed(p)
	if len(ops) != 1 || ops[0].Device == nil || ops[0].Target != "A1" || len(p.Topology.Transition) != 0 {
		t.Fatalf("%+v", ops)
	}
}

func TestResumeAppliesHeldChanges(t *testing.T) {
	c, s := pausedFixture(t, func(s *model.Snapshot) { s.Numbers["Patch.asio[0]"] = 0 })
	c.Intent.PauseSends, c.Intent.PauseDevices = true, true
	if p, _ := Build(c, s); p.HasChanges() {
		t.Fatal("paused plan has changes")
	}
	c.Intent.PauseSends, c.Intent.PauseDevices = false, false
	p, err := Build(c, s)
	if err != nil || !p.HasChanges() || len(p.Topology.Transition) == 0 || p.Topology.HeldSends != 0 {
		t.Fatal("resume did not plan the gated patch change", err)
	}
	applyPlan(&s, p)
	if p, err = Build(c, s); err != nil || p.HasChanges() {
		t.Fatal("resume did not converge", err)
	}
}
