//go:build windows

package mediasessions

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Client owns one companion handle. All methods must run on the OS thread
// that called Open, which the caller locks for the client's whole life. The
// DLL uses the Windows x64 C ABI; HRESULTs are signed int32.
type Client struct {
	dll                               *windows.DLL
	handle                            uintptr
	snapshot, art, command, closeProc *windows.Proc
	buffer                            []byte
}

// insufficientBuffer is HRESULT_FROM_WIN32(ERROR_INSUFFICIENT_BUFFER).
const insufficientBuffer = 0x8007007A

func result(code uintptr) error {
	if int32(code) < 0 {
		return fmt.Errorf("media sessions native error 0x%08X", uint32(code))
	}
	return nil
}

// Open loads only the explicitly supplied companion.
func Open(path string) (*Client, error) {
	dll, err := windows.LoadDLL(path)
	if err != nil {
		return nil, err
	}
	c := &Client{dll: dll, buffer: make([]byte, 64*1024)}
	var open *windows.Proc
	for name, target := range map[string]**windows.Proc{"MSOpen": &open, "MSSnapshot": &c.snapshot, "MSArt": &c.art, "MSCommand": &c.command, "MSClose": &c.closeProc} {
		if *target, err = dll.FindProc(name); err != nil {
			return nil, errors.Join(err, dll.Release())
		}
	}
	code, _, _ := open.Call(uintptr(unsafe.Pointer(&c.handle)))
	if err := result(code); err != nil {
		return nil, errors.Join(err, dll.Release())
	}
	return c, nil
}

// fetch calls a buffer-filling export, growing the buffer once if needed.
// It returns nil data for S_FALSE.
func (c *Client) fetch(call func(buf *byte, size uint32, needed *uint32) uintptr) ([]byte, error) {
	for attempt := 0; attempt < 2; attempt++ {
		var needed uint32
		code := call(&c.buffer[0], uint32(len(c.buffer)), &needed)
		if uint32(code) == insufficientBuffer && attempt == 0 && needed > uint32(len(c.buffer)) {
			c.buffer = make([]byte, needed)
			continue
		}
		if err := result(code); err != nil {
			return nil, err
		}
		if code == 1 { // S_FALSE: nothing to return.
			return nil, nil
		}
		return append([]byte(nil), c.buffer[:needed]...), nil
	}
	return nil, errors.New("media sessions result keeps growing")
}

// Snapshot lists the current sessions; IDs stay valid until the next call.
func (c *Client) Snapshot() ([]Session, error) {
	data, err := c.fetch(func(buf *byte, size uint32, needed *uint32) uintptr {
		code, _, _ := c.snapshot.Call(c.handle, uintptr(unsafe.Pointer(buf)), uintptr(size), uintptr(unsafe.Pointer(needed)))
		return code
	})
	if err != nil {
		return nil, err
	}
	return parseSnapshot(data)
}

// Art returns a session's thumbnail bytes (JPEG or PNG), or nil without one.
func (c *Client) Art(id string) ([]byte, error) {
	name, err := windows.BytePtrFromString(id)
	if err != nil {
		return nil, err
	}
	return c.fetch(func(buf *byte, size uint32, needed *uint32) uintptr {
		code, _, _ := c.art.Call(c.handle, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(buf)), uintptr(size), uintptr(unsafe.Pointer(needed)))
		return code
	})
}

// Command asks a session's player to act. Acceptance is not proof the player
// applied it; callers confirm with a later snapshot.
func (c *Client) Command(id string, op Op, value int64) error {
	name, err := windows.BytePtrFromString(id)
	if err != nil {
		return err
	}
	code, _, _ := c.command.Call(c.handle, uintptr(unsafe.Pointer(name)), uintptr(op), uintptr(value))
	if err := result(code); err != nil {
		return err
	}
	if code == 1 {
		return ErrDeclined
	}
	return nil
}

// Close releases the handle and the companion.
func (c *Client) Close() error {
	code, _, _ := c.closeProc.Call(c.handle)
	return errors.Join(result(code), c.dll.Release())
}
