package voicemeeter

import (
	"errors"
	"strings"
	"testing"
	"voice-snooter/internal/model"
)

type fakeAPI struct {
	login, refresh, editionCode, descCode, getCode, setCode int32
	edition                                                 int32
	logouts, releases                                       int
	lastParam, lastValue                                    string
}

func (a *fakeAPI) Login() int32            { return a.login }
func (a *fakeAPI) Logout() int32           { a.logouts++; return 0 }
func (a *fakeAPI) Refresh() int32          { return a.refresh }
func (a *fakeAPI) Edition() (int32, int32) { return a.edition, a.editionCode }
func (a *fakeAPI) Count(string) int32      { return 2 }
func (a *fakeAPI) Device(direction string, i int) (model.Device, int32) {
	driver := "wdm"
	if i == 1 {
		driver = "asio"
	}
	return model.Device{Name: "Device", Direction: direction, Driver: driver}, a.descCode
}
func (a *fakeAPI) Get(string) (string, int32)              { return "old", a.getCode }
func (a *fakeAPI) Set(p, v string) int32                   { a.lastParam = p; a.lastValue = v; return a.setCode }
func (a *fakeAPI) Release() error                          { a.releases++; return nil }
func (a *fakeAPI) GetNumber(string) (float32, int32)       { return 0, a.getCode }
func (a *fakeAPI) SetNumber(param string, value int) int32 { a.lastParam = param; return a.setCode }
func TestLoginLifecycle(t *testing.T) {
	for _, code := range []int32{0, 1} {
		a := &fakeAPI{login: code, edition: 2}
		c, e := connect(a)
		if e != nil {
			t.Fatal(e)
		}
		c.Close()
		c.Close()
		if a.logouts != 1 || a.releases != 1 {
			t.Fatal(a)
		}
		if _, e = c.Snapshot(); e == nil {
			t.Fatal("closed snapshot")
		}
	}
	a := &fakeAPI{login: -1}
	if _, e := connect(a); e == nil || a.releases != 1 {
		t.Fatal(e)
	}
}
func TestDisconnectedAndPartialSnapshots(t *testing.T) {
	a := &fakeAPI{edition: 2, refresh: -2}
	c, _ := connect(a)
	if _, e := c.Snapshot(); !errors.Is(e, ErrDisconnected) {
		t.Fatal(e)
	}
	a.refresh = 0
	a.descCode = -1
	s, e := c.Snapshot()
	if e == nil || len(s.Devices) > 0 {
		t.Fatal("partial enumeration escaped")
	}
	a.descCode = 0
	a.getCode = -3
	s, e = c.Snapshot()
	if e == nil || len(s.Devices) > 0 {
		t.Fatal("partial assignment escaped")
	}
}
func TestInventoryAndScopedSetter(t *testing.T) {
	a := &fakeAPI{edition: 2}
	c, _ := connect(a)
	s, e := c.Snapshot()
	if e != nil {
		t.Fatal(e)
	}
	if len(s.Devices) != 4 || len(s.Assignments) != 6 || s.Devices[1].Available || s.Devices[2].Direction != "output" {
		t.Fatal(s)
	}
	d := model.Device{Name: "Mic; literal name", Driver: "wdm", Direction: "input", Available: true}
	if e = c.Set("input:1", d); e != nil {
		t.Fatal(e)
	}
	if a.lastParam != "Strip[0].device.wdm" || a.lastValue != d.Name {
		t.Fatal(a)
	}
	if c.Set("input:4", d) == nil {
		t.Fatal("Banana accepted strip4")
	}
	d.Driver = "asio"
	if c.Set("input:1", d) == nil {
		t.Fatal("ASIO accepted")
	}
	if c.Set("garbage", d) == nil {
		t.Fatal("invalid slot accepted")
	}
}
func TestSignedStatus(t *testing.T) {
	if e := status("test", -5); e == nil || !strings.Contains(e.Error(), "-5") {
		t.Fatal(e)
	}
	if !errors.Is(status("test", -2), ErrDisconnected) {
		t.Fatal("lost disconnected status")
	}
}
