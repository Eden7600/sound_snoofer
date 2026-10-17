//go:build windows && amd64

package vroverlay

import (
	"testing"
	"unsafe"
)

func TestPinnedFunctionTableOffsets(t *testing.T) {
	var table overlayTable
	// Positions independently counted in Valve's IVROverlay_028 flat header.
	checks := []struct {
		offset uintptr
		slot   uintptr
	}{
		{unsafe.Offsetof(table.destroy), 3}, {unsafe.Offsetof(table.width), 22},
		{unsafe.Offsetof(table.visible), 45}, {unsafe.Offsetof(table.poll), 48},
		{unsafe.Offsetof(table.input), 50}, {unsafe.Offsetof(table.scale), 52},
		{unsafe.Offsetof(table.raw), 62}, {unsafe.Offsetof(table.create), 67},
		{unsafe.Offsetof(table.dashboard), 68}, {unsafe.Offsetof(table.active), 69},
	}
	for _, check := range checks {
		if check.offset != check.slot*8 {
			t.Fatalf("slot %d at %d", check.slot, check.offset)
		}
	}
	if interfaceVersion != "FnTable:IVROverlay_028" || unsafe.Sizeof(table) != 70*8 {
		t.Fatal("interface layout changed")
	}
	if table.valid() {
		t.Fatal("empty table accepted")
	}
	table.destroy, table.width, table.visible, table.poll, table.input, table.scale, table.raw, table.create, table.dashboard, table.active = 1, 1, 1, 1, 1, 1, 1, 1, 1, 1
	if !table.valid() {
		t.Fatal("complete table rejected")
	}
	table.poll = 0
	if table.valid() {
		t.Fatal("missing poll accepted")
	}
	if overlayResult("fake", 0) != nil || overlayResult("fake", 17) == nil {
		t.Fatal("native return code ignored")
	}
}
