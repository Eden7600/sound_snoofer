package voicemeeter

import (
	"strings"
	"testing"
)

type badBusAPI struct{ *fakeAPI }

func (a badBusAPI) GetNumber(p string) (float32, int32) {
	if strings.HasSuffix(p, ".B2") {
		return 0, -3
	}
	return 0, 0
}
func TestVoiceBusAPI(t *testing.T) {
	for _, edition := range []int32{2, 3} {
		a := &fakeAPI{edition: edition}
		c, _ := connect(a)
		s, e := c.Snapshot()
		if e != nil {
			t.Fatal(e)
		}
		if _, ok := s.Numbers["Strip[0].B2"]; !ok {
			t.Fatal("missing bus")
		}
		if e = c.SetNumber("Strip[0].B2", 1); e != nil {
			t.Fatal(e)
		}
		if e = c.SetNumber("Strip[0].B3", 1); (e == nil) != (edition == 3) {
			t.Fatal("edition bounds", e)
		}
		for _, p := range []string{"Strip[8].B2", "Strip[0].B4", "Strip[0].B2;Command.Restart"} {
			if c.SetNumber(p, 1) == nil {
				t.Fatal(p)
			}
		}
		if c.SetNumber("Strip[0].B2", 2) == nil {
			t.Fatal("nonboolean")
		}
		c.Close()
	}
	c, _ := connect(badBusAPI{&fakeAPI{edition: 3}})
	if _, e := c.Snapshot(); e == nil {
		t.Fatal("partial read accepted")
	}
}
