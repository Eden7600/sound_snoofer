package routing

import (
	"reflect"
	"testing"
	"voice-snooter/internal/config"
	"voice-snooter/internal/model"
)

func setup(t *testing.T) (config.Config, model.Snapshot) {
	t.Helper()
	c, e := config.Decode([]byte(`{"version":1,"routes":[{"target":"A1","candidates":[{"driver":"wdm","pattern":"(?i)airpods"},{"driver":"wdm","pattern":"(?i)steelseries"}]},{"target":"input:1","candidates":[{"driver":"wdm","pattern":"(?i)volt.*2"},{"driver":"wdm","pattern":"(?i)webcam"}]}]}`))
	if e != nil {
		t.Fatal(e)
	}
	s := model.Snapshot{Edition: 2, Assignments: map[string]string{"input:1": "old mic", "A1": "old speakers"}, Devices: []model.Device{
		{Name: "INPUT 1/2 (Volt 2)", Direction: "input", Driver: "wdm", Available: true},
		{Name: "webcam mic", Direction: "input", Driver: "wdm", Available: true},
		{Name: "AirPods stereo", Direction: "output", Driver: "wdm", Available: true},
		{Name: "SteelSeries speakers", Direction: "output", Driver: "wdm", Available: true},
	}}
	return c, s
}
func TestPriorityFallbackAndReconnect(t *testing.T) {
	c, s := setup(t)
	p, e := Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	if p.Decisions[0].Target != "input:1" || p.Decisions[0].Desired.Name != "INPUT 1/2 (Volt 2)" || p.Decisions[1].Desired.Name != "AirPods stereo" {
		t.Fatal(p)
	}
	original := p.Key()
	s.Devices[0].Available = false
	s.Devices[2].Available = false
	p, e = Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	if p.Decisions[0].Desired.Name != "webcam mic" || p.Decisions[1].Desired.Name != "SteelSeries speakers" {
		t.Fatal(p)
	}
	s.Devices[0].Available = true
	s.Devices[2].Available = true
	p, _ = Build(c, s)
	if p.Key() != original {
		t.Fatal("did not return to preferred devices")
	}
}
func TestAmbiguityAndDirectionScoping(t *testing.T) {
	c, s := setup(t)
	s.Devices = append(s.Devices, model.Device{Name: "Volt 2 second", Direction: "input", Driver: "wdm", Available: true}, model.Device{Name: "AirPods mic", Direction: "input", Driver: "wdm", Available: true})
	p, e := Build(c, s)
	if e != nil {
		t.Fatal(e)
	}
	if p.Decisions[0].Desired.Name != "webcam mic" || p.Decisions[1].Desired.Name != "AirPods stereo" {
		t.Fatal(p)
	}
}
func TestDriverScoping(t *testing.T) {
	c, s := setup(t)
	s.Devices[0].Driver = "asio"
	p, _ := Build(c, s)
	if p.Decisions[0].Desired.Name != "webcam mic" {
		t.Fatal(p)
	}
}
func TestNoCandidatesAndUnchanged(t *testing.T) {
	c, s := setup(t)
	s.Devices = nil
	p, _ := Build(c, s)
	if p.HasChanges() || !p.HasUnresolved() {
		t.Fatal(p)
	}
	for _, d := range p.Decisions {
		if d.Desired != nil {
			t.Fatal(d)
		}
	}
	c, s = setup(t)
	s.Assignments["input:1"] = s.Devices[0].Name
	s.Assignments["A1"] = s.Devices[2].Name
	p, _ = Build(c, s)
	if p.HasChanges() || p.HasUnresolved() {
		t.Fatal(p)
	}
}
func TestOrderIndependentAndMissingSnapshot(t *testing.T) {
	c, s := setup(t)
	p, _ := Build(c, s)
	s.Devices[0], s.Devices[3] = s.Devices[3], s.Devices[0]
	q, _ := Build(c, s)
	if !reflect.DeepEqual(p, q) {
		t.Fatal("enumeration order changed plan")
	}
	delete(s.Assignments, "A1")
	if _, e := Build(c, s); e == nil {
		t.Fatal("accepted partial snapshot")
	}
}
func TestPotatoSlots(t *testing.T) {
	c, s := setup(t)
	c.Routes[0].Target = "A5"
	s.Assignments["A5"] = ""
	if _, e := Build(c, s); e == nil {
		t.Fatal("Banana accepted A5")
	}
	s.Edition = 3
	if _, e := Build(c, s); e != nil {
		t.Fatal(e)
	}
}
