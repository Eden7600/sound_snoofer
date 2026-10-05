package voicemeeter

import (
	"testing"
	"time"
)

func TestSingleFreshHostEnumeration(t *testing.T) {
	c, e := connect(&fakeAPI{edition: 3})
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	calls := 0
	c.processList = func() ([]string, error) { calls++; return []string{"element.exe", "vrserver.exe"}, nil }
	started := time.Now()
	s, e := c.Snapshot()
	if e != nil {
		t.Fatal(e)
	}
	if calls != 1 || !s.Element.Running || s.SteamVR != nil {
		t.Fatal("duplicate/missing host observation", calls)
	}
	_, e = c.ParameterSnapshot()
	if e != nil || calls != 2 {
		t.Fatal("fast snapshot reused stale host list", calls, e)
	}
	for n := 0; n < 1000; n++ {
		if _, e = c.ParameterSnapshot(); e != nil {
			t.Fatal(e)
		}
	}
	if calls != 1002 {
		t.Fatal("fresh host enumeration count", calls)
	}
	t.Logf("1002 fake observations: %s; process enumeration calls 1002 (previous flow 2004)", time.Since(started))
}
