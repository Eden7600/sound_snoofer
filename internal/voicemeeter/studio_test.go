package voicemeeter

import (
	"testing"
	"sound-snoofer/internal/model"
)

func TestASIOAndNumericWriteBoundaries(t *testing.T) {
	a := &fakeAPI{edition: 2}
	c, _ := connect(a)
	d := model.Device{Name: "Volt", Direction: "output", Driver: "asio", Available: true}
	if e := c.Set("A1", d); e != nil {
		t.Fatal(e)
	}
	if a.lastParam != "Bus[0].device.asio" {
		t.Fatal(a.lastParam)
	}
	if c.Set("A2", d) == nil {
		t.Fatal("ASIO outside A1")
	}
	for _, param := range []string{"Patch.asio[0]", "Patch.asio[3]", "Strip[3].A2"} {
		if e := c.SetNumber(param, 1); e != nil {
			t.Fatal(e)
		}
	}
	for _, param := range []string{"Patch.asio[4]", "Strip[5].A1", "Strip[0].A4", "Strip[0].Gain", "Command.Restart", "Strip[0].A1;Command.Restart"} {
		if c.SetNumber(param, 1) == nil {
			t.Fatal("unmanaged write accepted", param)
		}
	}
	if c.SetNumber("Patch.asio[0]", 3) == nil || c.SetNumber("Strip[3].A2", 2) == nil {
		t.Fatal("invalid numeric value")
	}
	a.setCode = -3
	if c.SetNumber("Patch.asio[0]", 1) == nil {
		t.Fatal("ignored failure")
	}
}
