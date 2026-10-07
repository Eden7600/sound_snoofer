//go:build windows

package discord

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"golang.org/x/sys/windows"
)

// pipe is a Discord IPC named pipe opened for overlapped I/O, so a pending
// read on the reader goroutine never blocks the worker's writes. Close
// cancels pending I/O.
type pipe struct {
	handle windows.Handle
	once   sync.Once
}

// errNoDiscord means no Discord IPC pipe accepted a connection.
var errNoDiscord = errors.New("Discord closed")

// dialPipe connects to the first of \\.\pipe\discord-ipc-0 to -9 that is open.
func dialPipe() (io.ReadWriteCloser, string, error) {
	for n := 0; n < 10; n++ {
		name := fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, n)
		path, err := windows.UTF16PtrFromString(name)
		if err != nil {
			return nil, "", err
		}
		h, err := windows.CreateFile(path, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OVERLAPPED, 0)
		if err == nil {
			return &pipe{handle: h}, name, nil
		}
	}
	return nil, "", errNoDiscord
}

// io runs one overlapped read or write and waits for it.
func (p *pipe) io(buf []byte, read bool) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(event)
	// The kernel writes the OVERLAPPED asynchronously; heap memory never moves.
	o := &windows.Overlapped{HEvent: event}
	done := new(uint32)
	if read {
		err = windows.ReadFile(p.handle, buf, done, o)
	} else {
		err = windows.WriteFile(p.handle, buf, done, o)
	}
	if err != nil && err != windows.ERROR_IO_PENDING {
		return 0, err
	}
	if err = windows.GetOverlappedResult(p.handle, o, done, true); err != nil {
		if err == windows.ERROR_OPERATION_ABORTED || err == windows.ERROR_BROKEN_PIPE {
			return int(*done), io.EOF
		}
		return int(*done), err
	}
	return int(*done), nil
}

func (p *pipe) Read(buf []byte) (int, error) {
	return p.io(buf, true)
}

func (p *pipe) Write(buf []byte) (int, error) {
	written := 0
	for written < len(buf) {
		n, err := p.io(buf[written:], false)
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

// Close cancels pending reads and writes, then closes the handle.
func (p *pipe) Close() error {
	var err error
	p.once.Do(func() {
		_ = windows.CancelIoEx(p.handle, nil) // Nothing pending is not an error worth reporting.
		err = windows.CloseHandle(p.handle)
	})
	return err
}
