package voicemeeter

import (
	"math"
	"testing"
)

type levelAPI struct {
	fakeAPI
	calls   [][2]int
	values  map[[2]int]float32
	failure [2]int
}

func (a *levelAPI) GetLevel(kind, channel int) (float32, int32) {
	key := [2]int{kind, channel}
	a.calls = append(a.calls, key)
	if key == a.failure {
		return 0, -3
	}
	return a.values[key], 0
}
func TestGainLevelsMappingAndFailures(t *testing.T) {
	a := &levelAPI{fakeAPI: fakeAPI{edition: 3}, values: map[[2]int]float32{{3, 7}: 0.5, {3, 8}: 0.25, {2, 7}: 1.2}}
	c, _ := connect(a)
	defer c.Close()
	levels := c.GainLevels(3)
	if levels["Bus[0].Gain"] != 0.5 || levels["Bus[1].Gain"] != 0.25 || levels["Strip[3].Gain"] != 1.2 {
		t.Fatal(levels)
	}
	if len(a.calls) != 18 || a.calls[16] != [2]int{2, 6} || a.calls[17] != [2]int{2, 7} {
		t.Fatal(a.calls)
	}
	a.failure = [2]int{3, 7}
	a.values[[2]int{3, 8}] = float32(math.NaN())
	levels = c.GainLevels(3)
	if _, ok := levels["Bus[0].Gain"]; ok {
		t.Fatal("failed read became silence")
	}
	if _, ok := levels["Bus[1].Gain"]; ok {
		t.Fatal("NaN was published")
	}
	for _, v := range []float32{-1, float32(math.Inf(1))} {
		a.values[[2]int{3, 8}] = v
		if _, ok := c.GainLevels(3)["Bus[1].Gain"]; ok {
			t.Fatal("invalid level accepted", v)
		}
	}
	a.failure = [2]int{-1, -1}
	a.values = nil
	levels = c.GainLevels(-1)
	if len(levels) != 2 || levels["Bus[0].Gain"] != 0 {
		t.Fatal("silence/off", levels)
	}
	a.edition = 2
	a.calls = nil
	if len(c.GainLevels(3)) != 2 || len(a.calls) != 16 {
		t.Fatal("invalid Banana strip read")
	}
	a.calls = nil
	if len(c.GainLevels(2)) != 3 || a.calls[16] != [2]int{2, 4} {
		t.Fatal("Banana physical channel mapping", a.calls)
	}
	a.refresh = -2
	if c.GainLevels(0) != nil {
		t.Fatal("disconnected retained meter")
	}
}
