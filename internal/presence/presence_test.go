package presence

import "testing"

func TestApply(t *testing.T) {
	s := State{Known: true}
	steps := []struct {
		msg, wparam uintptr
		display     uint32
		changed     bool
		away        bool
	}{
		{wmWTSSessionChange, wtsSessionLock, 0, true, true},
		{wmPowerBroadcast, pbtPowerSettingChange, 0, true, true},
		{wmWTSSessionChange, wtsSessionUnlock, 0, true, true}, // Monitors still off.
		{wmPowerBroadcast, pbtPowerSettingChange, 2, true, false},
		{wmPowerBroadcast, pbtPowerSettingChange, 1, true, false},
		{wmWTSSessionChange, 5, 0, false, false}, // Logon and other changes.
		{wmPowerBroadcast, 0x000A, 0, false, false},
		{0x0010, 0, 0, false, false},
	}
	for n, step := range steps {
		var changed bool
		s, changed = apply(s, step.msg, step.wparam, step.display)
		if changed != step.changed || s.Away() != step.away {
			t.Fatalf("step %d: changed %v away %v", n, changed, s.Away())
		}
	}
	if (State{Locked: true, DisplayOff: true}).Away() {
		t.Fatal("an unknown state must never hide the console")
	}
}
