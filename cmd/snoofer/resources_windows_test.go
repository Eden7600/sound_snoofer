//go:build windows && amd64

package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

func TestApplicationIconMatchesTrayAsset(t *testing.T) {
	source, err := os.ReadFile("../../app/tray.ico")
	if err != nil {
		t.Fatal(err)
	}
	module, _, err := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetModuleHandleW").Call(0)
	if module == 0 {
		t.Fatal(err)
	}
	resource := func(id, kind uint16) []byte {
		t.Helper()
		handle, err := windows.FindResource(windows.Handle(module), windows.ResourceID(id), windows.ResourceID(kind))
		if err != nil {
			t.Fatalf("missing resource %d/%d: %v", kind, id, err)
		}
		data, err := windows.LoadResourceData(windows.Handle(module), handle)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	group := resource(1, 14) // RT_GROUP_ICON
	if len(source) < 6 || len(group) < 6 || !bytes.Equal(source[:6], group[:6]) {
		t.Fatal("executable and tray icon headers differ")
	}
	count := int(binary.LittleEndian.Uint16(source[4:6]))
	if count == 0 || len(group) < 6+14*count || len(source) < 6+16*count {
		t.Fatal("invalid icon directory")
	}
	for i := 0; i < count; i++ {
		entry := source[6+16*i : 6+16*(i+1)]
		embedded := group[6+14*i : 6+14*(i+1)]
		if !bytes.Equal(entry[:12], embedded[:12]) {
			t.Fatalf("icon frame %d metadata differs", i)
		}
		size := int(binary.LittleEndian.Uint32(entry[8:12]))
		offset := int(binary.LittleEndian.Uint32(entry[12:16]))
		if offset < 0 || size < 0 || offset+size > len(source) {
			t.Fatal("invalid ICO frame bounds")
		}
		id := binary.LittleEndian.Uint16(embedded[12:14])
		if !bytes.Equal(source[offset:offset+size], resource(id, 3)) {
			t.Fatalf("icon frame %d differs; regenerate Windows resources", i)
		}
	}
	load := windows.NewLazySystemDLL("user32.dll").NewProc("LoadImageW")
	for _, size := range []uintptr{16, 32, 256} {
		icon, _, err := load.Call(module, 1, 1, size, size, 0x8000)
		if icon == 0 {
			t.Fatalf("Windows cannot load %dpx application icon: %v", size, err)
		}
		// LR_SHARED resource handles are released by Windows at process exit.
	}
}
