// Package presence observes whether someone can be at the console: whether
// the Windows session is locked and whether the monitors are on.
package presence

// State is the latest observation. Known is false when Windows does not
// report it, which callers treat as present.
type State struct {
	Known      bool
	Locked     bool
	DisplayOff bool
}

// Away reports whether nobody should be able to see or use the console.
func (s State) Away() bool { return s.Known && (s.Locked || s.DisplayOff) }

// Window messages and values the watcher handles.
const (
	wmWTSSessionChange     = 0x02B1
	wmPowerBroadcast       = 0x0218
	wtsSessionLock         = 7
	wtsSessionUnlock       = 8
	pbtPowerSettingChange  = 0x8013
	displayOff             = 0
	sessionFlagLock        = 0
	sessionFlagUnlock      = 1
	powerSettingDataOffset = 20 // After the GUID and its DWORD length.
)

// apply updates s from one window message. For WM_POWERBROADCAST, display
// is the console display state carried by the power setting (0 off, 1 on,
// 2 dimmed); ok reports whether the message changed anything it observes.
func apply(s State, msg, wparam uintptr, display uint32) (State, bool) {
	switch {
	case msg == wmWTSSessionChange && wparam == wtsSessionLock:
		s.Locked = true
	case msg == wmWTSSessionChange && wparam == wtsSessionUnlock:
		s.Locked = false
	case msg == wmPowerBroadcast && wparam == pbtPowerSettingChange:
		// Dimmed monitors still show the desktop.
		s.DisplayOff = display == displayOff
	default:
		return s, false
	}
	return s, true
}
