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
	dll                          *windows.DLL
	start, stop, read            *windows.Proc
	enabled, active, unsafeClose bool
	err                          error
}

func (a *winAPI) SetMonitoring(enable bool) error {
	m := &a.monitor
	if m.unsafeClose {
		return fmt.Errorf("callback cleanup uncertain; restart Sound Snoofer")
	}
	if enable == m.enabled {
		return m.err
	}
	m.enabled = enable
	if !enable {
		if m.active {
			code, _, _ := m.stop.Call()
			if result(code) != 0 {
				m.unsafeClose = true
				m.err = fmt.Errorf("callback cleanup returned %d; restart Sound Snoofer", result(code))
				return m.err
			}
		}
		m.active = false
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
		for name, target := range map[string]**windows.Proc{"SnooferStart": &m.start, "SnooferStop": &m.stop, "SnooferRead": &m.read} {
			*target, m.err = m.dll.FindProc(name)
			if m.err != nil {
				// No registration occurred. Keep the error visible until a new enable attempt.
				_ = m.dll.Release()
				m.dll = nil
				return m.err
			}
		}
	}
	code, _, _ := m.start.Call(uintptr(a.dll.Handle))
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
