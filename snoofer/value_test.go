package snoofer

import "testing"

func TestOnOff(t *testing.T) {
	for _, tc := range []struct {
		value bool
		want  string
	}{{true, "On"}, {false, "Off"}} {
		if got := OnOff(tc.value); got != tc.want {
			t.Errorf("OnOff(%t) = %q; want %q", tc.value, got, tc.want)
		}
	}
}
