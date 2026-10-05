//go:build windows && amd64

package process

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

func Names() ([]string, error) {
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	// Closing a read-only enumeration handle has no recoverable failure action.
	defer windows.CloseHandle(handle)
	var currentSession uint32
	if err = windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &currentSession); err != nil {
		return nil, err
	}
	entry := windows.ProcessEntry32{}
	entry.Size = uint32(unsafe.Sizeof(entry))
	names := []string{}
	err = windows.Process32First(handle, &entry)
	for err == nil {
		var processSession uint32
		if windows.ProcessIdToSessionId(entry.ProcessID, &processSession) == nil && currentSession == processSession {
			names = append(names, windows.UTF16ToString(entry.ExeFile[:]))
		}
		err = windows.Process32Next(handle, &entry)
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return nil, err
	}
	return names, nil
}
