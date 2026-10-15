package audio

import "testing"

func TestEchoPlaybackBusNames(t *testing.T) {
	for target, firstChannel := range map[string]int{
		"A1": 0, "A2": 8, "A5": 32, "A01": 0, "A+1": 0,
		"": -1, "A": -1, "A0": -1, "A6": -1, "B1": -1, "a1": -1,
		"A-1": -1, "A 1": -1, "A1 ": -1, "A1x": -1,
	} {
		got := echoTargets(echoState(0, "desk", target))
		if got.Reference[0] != firstChannel || (got.Reason != "") != (firstChannel < 0) {
			t.Errorf("echoTargets(%q) = %+v; want first channel %d", target, got, firstChannel)
		}
	}
}
