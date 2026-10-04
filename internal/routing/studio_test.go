package routing

import (
	"fmt"
	"testing"
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

func studioFixture(t *testing.T) (config.Config, model.Snapshot) {
	t.Helper()
	c, e := config.Decode([]byte(`{"version":1,"studio":{"asio_pattern":"^Volt ASIO$","presence_pattern":"^Volt input$","playback":[{"driver":"wdm","pattern":"AirPods"},{"driver":"wdm","pattern":"speakers"}],"fallback_mic":[{"driver":"wdm","pattern":"webcam"}],"move_playback_routing":true,"playback_sources":["virtual:1"]}}`))
	if e != nil {
		t.Fatal(e)
	}
	s := model.Snapshot{Edition: 2, Assignments: map[string]string{}, Numbers: map[string]float32{}, Devices: []model.Device{{Name: "Volt ASIO", Direction: "output", Driver: "asio"}, {Name: "Volt input", Direction: "input", Driver: "wdm", Available: true}, {Name: "webcam", Direction: "input", Driver: "wdm", Available: true}, {Name: "speakers", Direction: "output", Driver: "wdm", Available: true}}}
	for _, slot := range model.Slots(2) {
		s.Assignments[slot] = ""
	}
	s.Assignments["A1"] = "speakers"
	for i := 0; i < 4; i++ {
		s.Numbers[fmt.Sprintf("Patch.asio[%d]", i)] = 0
	}
	for i := 0; i < 8; i++ {
		for j := 1; j <= 5; j++ {
			s.Numbers[fmt.Sprintf("Strip[%d].A%d", i, j)] = 0
		}
	}
	s.Numbers["Strip[3].A1"] = 1
	return c, s
}
func TestStudioVoltTopology(t *testing.T) {
	c, s := studioFixture(t)
	p, e := Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	if !p.Topology.ASIOActive || p.Topology.PlaybackTarget != "A2" {
		t.Fatal(p)
	}
	want := map[string]int{"Patch.asio[0]": 1, "Patch.asio[1]": 1, "Patch.asio[2]": 2, "Patch.asio[3]": 2, "Strip[3].A1": 0, "Strip[3].A2": 1}
	for _, op := range p.Topology.Operations {
		if op.Parameter != "" {
			if v, ok := want[op.Parameter]; ok {
				if v != op.Value {
					t.Fatal(op)
				}
				delete(want, op.Parameter)
			}
		}
		if op.Target == "input:1" || op.Target == "input:2" {
			if op.Device.Name != "" {
				t.Fatal("ASIO inputs must not open WDM", op)
			}
		}
	}
	if len(want) > 0 {
		t.Fatal(want)
	}
}
func TestInstalledASIODriverIsNotPresence(t *testing.T) {
	c, s := studioFixture(t)
	s.Devices[1].Available = false
	s.Assignments["A1"] = "Volt ASIO"
	s.Assignments["A2"] = "speakers"
	p, e := Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	if p.Topology.ASIOActive || p.Topology.PlaybackTarget != "A1" {
		t.Fatal(p)
	}
	found := false
	for _, op := range p.Topology.Operations {
		if op.Target == "input:1" {
			found = op.Device.Name == "webcam"
		}
		if op.Parameter == "Patch.asio[2]" && op.Value != 0 {
			t.Fatal(op)
		}
	}
	if !found {
		t.Fatal("no webcam fallback")
	}
}
func TestLowestFreeAndOccupiedOutputs(t *testing.T) {
	c, s := studioFixture(t)
	s.Assignments["A2"] = "unmanaged monitor"
	p, e := Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	if p.Topology.PlaybackTarget != "A3" {
		t.Fatal(p)
	}
	for _, op := range p.Topology.Operations {
		if op.Target == "A2" {
			t.Fatal("overwrites occupied bus")
		}
	}
	s.Assignments["A3"] = "other monitor"
	if _, e = Build(c, s); e == nil {
		t.Fatal("accepted no free output")
	}
	s.Assignments["A1"] = "unmanaged interface"
	if _, e = Build(c, s); e == nil {
		t.Fatal("overwrites reserved A1")
	}
}
func TestPrimaryVirtualRuleRepairsDrift(t *testing.T) {
	c, s := studioFixture(t)
	s.Assignments["A1"] = "Volt ASIO"
	s.Assignments["A2"] = "speakers"
	for i, v := range []float32{1, 1, 2, 2} {
		s.Numbers[fmt.Sprintf("Patch.asio[%d]", i)] = v
	}
	s.Numbers["Strip[3].A2"] = 0
	p, e := Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, op := range p.Topology.Operations {
		if op.Parameter == "Strip[3].A2" {
			found = op.Value == 1 && op.Change
		}
	}
	if !found || !p.HasChanges() {
		t.Fatal("rule did not repair drift")
	}
}
func TestPotatoSemanticVirtualSource(t *testing.T) {
	c, s := studioFixture(t)
	s.Edition = 3
	for _, slot := range model.Slots(3) {
		if _, ok := s.Assignments[slot]; !ok {
			s.Assignments[slot] = ""
		}
	}
	p, e := Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, op := range p.Topology.Operations {
		if op.Parameter == "Strip[5].A2" && op.Value == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("Potato VAIO index is not 5")
	}
}
func TestAmbiguousPresenceMissingPlayback(t *testing.T) {
	c, s := studioFixture(t)
	s.Devices = append(s.Devices, s.Devices[1])
	if _, e := Build(c, s); e == nil {
		t.Fatal("ambiguous presence accepted")
	}
	c, s = studioFixture(t)
	s.Devices[3].Available = false
	p, e := Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	if !p.HasUnresolved() || len(p.Topology.Operations) != 0 {
		t.Fatal("changed ASIO without playback destination")
	}
}
