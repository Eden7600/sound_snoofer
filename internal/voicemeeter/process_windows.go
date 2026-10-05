//go:build windows && amd64

package voicemeeter

import (
	"errors"
	"golang.org/x/sys/windows"
	"strings"
	"unsafe"

	"sound-snoofer/internal/model"
)

func elementProcess() model.ProcessStatus { return observeElement(processNames) }
func processNames() ([]string, error) {
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	// Closing a read-only enumeration handle has no recoverable failure action.
	defer windows.CloseHandle(handle)
	entry := windows.ProcessEntry32{}
	entry.Size = uint32(unsafe.Sizeof(entry))
	names := []string{}
	err = windows.Process32First(handle, &entry)
	for err == nil {
		var currentSession, processSession uint32
		if windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &currentSession) == nil && windows.ProcessIdToSessionId(entry.ProcessID, &processSession) == nil && currentSession == processSession {
			names = append(names, windows.UTF16ToString(entry.ExeFile[:]))
		}
		err = windows.Process32Next(handle, &entry)
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return nil, err
	}
	return names, nil
}

func steamVRProcess() model.ProcessStatus {
	names, err := processNames()
	if err != nil {
		return model.ProcessStatus{Error: err.Error()}
	}
	for _, name := range names {
		if strings.EqualFold(name, "vrserver.exe") {
			return model.ProcessStatus{Known: true, Running: true}
		}
	}
	return model.ProcessStatus{Known: true}
}
