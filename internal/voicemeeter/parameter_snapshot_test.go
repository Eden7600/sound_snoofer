package voicemeeter

import "testing"

type countedAPI struct {
	*fakeAPI
	enumerations int
}

func (a *countedAPI) Count(s string) int32 { a.enumerations++; return a.fakeAPI.Count(s) }
func TestParameterSnapshotSkipsEnumeration(t *testing.T) {
	a := &countedAPI{fakeAPI: &fakeAPI{edition: 3}}
	c, _ := connect(a)
	defer c.Close()
	if _, err := c.ParameterSnapshot(); err == nil {
		t.Fatal("missing inventory accepted")
	}
	s, err := c.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	count := a.enumerations
	s.Devices[0].Name = "mutated outside"
	fast, err := c.ParameterSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if a.enumerations != count || fast.Devices[0].Name == s.Devices[0].Name {
		t.Fatal("enumerated or shared inventory")
	}
	a.refresh = -2
	if _, err = c.ParameterSnapshot(); err == nil {
		t.Fatal("disconnect ignored")
	}
}
