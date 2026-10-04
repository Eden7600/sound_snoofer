package controller

import (
	"context"
	"fmt"
	"testing"
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

type topologyBackend struct {
	*fakeBackend
	numbers    int
	failNumber bool
}

func (b *topologyBackend) SetNumber(param string, value int) error {
	b.numbers++
	if b.failNumber {
		return fmt.Errorf("numeric failure")
	}
	b.s.Numbers[param] = float32(value)
	return nil
}
func topologyFixture(t *testing.T) (*Controller, *topologyBackend) {
	t.Helper()
	c, b, _ := fixture(t)
	cfg, e := config.Decode([]byte(`{"version":1,"studio":{"asio_pattern":"^Volt ASIO$","presence_pattern":"^Volt$","playback":[{"driver":"wdm","pattern":"AirPods"},{"driver":"wdm","pattern":"speakers"}],"fallback_mic":[{"driver":"wdm","pattern":"webcam"}],"move_playback_routing":true,"playback_sources":["virtual:1"]}}`))
	if e != nil {
		t.Fatal(e)
	}
	c.Config = cfg
	b.s.Assignments["A2"] = ""
	b.s.Assignments["A3"] = ""
	b.s.Assignments["input:2"] = ""
	b.s.Assignments["input:3"] = "unmanaged mic"
	b.s.Devices = append(b.s.Devices, model.Device{Name: "Volt ASIO", Driver: "asio", Direction: "output"})
	b.s.Numbers = map[string]float32{}
	for i := 0; i < 4; i++ {
		b.s.Numbers[fmt.Sprintf("Patch.asio[%d]", i)] = 0
	}
	for i := 0; i < 5; i++ {
		for j := 1; j <= 3; j++ {
			b.s.Numbers[fmt.Sprintf("Strip[%d].A%d", i, j)] = 0
		}
	}
	b.s.Numbers["Strip[3].A1"] = 1
	b.s.Numbers["Strip[0].A1"] = 1
	b.s.Numbers["Strip[2].A3"] = 1
	tb := &topologyBackend{fakeBackend: b}
	c.Backend = tb
	return c, tb
}
func TestTopologyRoundTripAndRules(t *testing.T) {
	c, b := topologyFixture(t)
	p, e := c.Plan()
	if e != nil {
		t.Fatal(e)
	}
	if e = c.Apply(context.Background(), p); e != nil {
		t.Fatal(e)
	}
	if b.s.Assignments["A1"] != "Volt ASIO" || b.s.Assignments["A2"] != "AirPods" || b.s.Numbers["Patch.asio[2]"] != 2 || b.s.Numbers["Strip[3].A2"] != 1 || b.s.Numbers["Strip[3].A1"] != 0 || b.s.Numbers["Strip[0].A2"] != 1 {
		t.Fatal(b.s)
	}
	if b.s.Assignments["input:3"] != "unmanaged mic" || b.s.Numbers["Strip[2].A3"] != 1 {
		t.Fatal("unmanaged routing changed")
	}
	writes := len(b.writes) + b.numbers
	p, e = c.Plan()
	if e != nil || p.HasChanges() {
		t.Fatal(e, p)
	}
	if e = c.Apply(context.Background(), p); e != nil {
		t.Fatal(e)
	}
	if len(b.writes)+b.numbers != writes {
		t.Fatal("not idempotent")
	}
	b.s.Numbers["Strip[3].A2"] = 0
	p, _ = c.Plan()
	if e = c.Apply(context.Background(), p); e != nil {
		t.Fatal(e)
	}
	if b.s.Numbers["Strip[3].A2"] != 1 {
		t.Fatal("did not enforce rule")
	}
	b.s.Devices[0].Available = false
	p, e = c.Plan()
	if e != nil {
		t.Fatal(e)
	}
	if e = c.Apply(context.Background(), p); e != nil {
		t.Fatal(e)
	}
	if b.s.Assignments["A1"] != "AirPods" || b.s.Assignments["A2"] != "" || b.s.Assignments["input:1"] != "webcam" || b.s.Numbers["Patch.asio[2]"] != 0 || b.s.Numbers["Strip[3].A1"] != 1 || b.s.Numbers["Strip[3].A2"] != 0 {
		t.Fatal(b.s)
	}
	b.s.Devices[0].Available = true
	p, _ = c.Plan()
	if e = c.Apply(context.Background(), p); e != nil {
		t.Fatal(e)
	}
}
func TestTopologyStopsOnNumericFailure(t *testing.T) {
	c, b := topologyFixture(t)
	b.failNumber = true
	p, _ := c.Plan()
	if e := c.Apply(context.Background(), p); e == nil {
		t.Fatal("ignored failed patch")
	}
	if b.numbers != 1 {
		t.Fatal("continued after failed numeric write")
	}
}
