//go:build windows && amd64

package voicemeeter

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
	"sound-snoofer/internal/model"
)

type callbackMonitor struct {
	dll                             *windows.DLL
	start, stop, read, insertStages *windows.Proc
	enabled, active, unsafeClose    bool
	insert                          *InsertHook // Stages of the current registration.
	err                             error
}

// SetCallback registers while monitoring or an insert is wanted. Insert stages
// change only between registrations, so a hook change stops, unregisters,
// registers and starts again. Once this returns nil after removing a hook,
// the monitor no longer calls it.
func (a *winAPI) SetCallback(monitor bool, insert *InsertHook) error {
	m := &a.monitor
	if m.unsafeClose {
		return fmt.Errorf("callback cleanup uncertain; restart Sound Snoofer")
	}
	enable := monitor || insert != nil
	if enable == m.enabled && sameHook(insert, m.insert) {
		return m.err
	}
	if m.active {
		code, _, _ := m.stop.Call()
		if result(code) != 0 {
			m.unsafeClose = true
			m.err = fmt.Errorf("callback cleanup returned %d; restart Sound Snoofer", result(code))
			return m.err
		}
		m.active = false
	}
	m.enabled = enable
	m.insert = nil
	if insert != nil {
		copied := *insert
		m.insert = &copied
	}
	if !enable {
		m.err = nil
		return nil
	}
	if m.dll == nil {
		exe, err := os.Executable()
		if err != nil {
			m.err = err
			return err
		}
		path := filepath.Join(filepath.Dir(exe), "snoofer-audio-monitor.dll")
		m.dll, m.err = windows.LoadDLL(path)
		if m.err != nil {
			m.err = fmt.Errorf("audio monitor missing or unloadable beside executable: %w", m.err)
			return m.err
		}
		procs := map[string]**windows.Proc{"SnooferStart": &m.start, "SnooferStop": &m.stop, "SnooferRead": &m.read, "SnooferSetInsert": &m.insertStages}
		for name, target := range procs {
			*target, m.err = m.dll.FindProc(name)
			if m.err != nil {
				// No registration occurred. Keep the error visible until a new enable attempt.
				_ = m.dll.Release()
				m.dll = nil
				return m.err
			}
		}
	}
	var stages InsertHook
	if m.insert != nil {
		stages = *m.insert
	}
	code, _, _ := m.insertStages.Call(stages.Input, stages.Output, stages.Context)
	if result(code) != 0 {
		m.err = fmt.Errorf("callback insert stages refused with %d", result(code))
		return m.err
	}
	code, _, _ = m.start.Call(uintptr(a.dll.Handle))
	if result(code) != 0 {
		if result(code) == -12 {
			m.unsafeClose = true
		}
		m.err = fmt.Errorf("callback registration/start returned %d (1: slot owned by another app)", result(code))
		return m.err
	}
	m.active = true
	m.err = nil
	return nil
}

func (a *winAPI) CallbackStatus() *model.CallbackStatus {
	m := &a.monitor
	if !m.enabled && !m.unsafeClose {
		return nil
	}
	s := &model.CallbackStatus{Active: m.active}
	if m.err != nil {
		s.Error = m.err.Error()
	}
	if m.active {
		var values [7]uint32
		m.read.Call(uintptr(unsafe.Pointer(&values[0])))
		s.Buffers, s.Synced, s.Starting, s.Ending, s.Changes = values[0], values[1], values[2], values[3], values[4]
		if values[5] != 0 || values[6] != 0 {
			s.Error = "invalid or unexpected audio callback; disable automatic recovery"
		}
	}
	return s
}

func sameHook(a, b *InsertHook) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
