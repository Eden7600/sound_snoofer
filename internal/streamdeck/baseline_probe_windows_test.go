//go:build windows

package streamdeck

import (
	"fmt"
	"os"
	"testing"
	"unsafe"
)

// probeInitialInput preserves the historical query outside the production path.
func probeInitialInput(d *device) ([]byte, error) {
	report := make([]byte, d.input)
	report[0] = 1
	r, _, err := hid.NewProc("HidD_GetInputReport").Call(uintptr(d.handle), uintptr(unsafe.Pointer(&report[0])), uintptr(len(report)))
	if r == 0 {
		return nil, fmt.Errorf("read Stream Deck current input: %w", err)
	}
	return report, nil
}

func TestNativeInitialInput(t *testing.T) {
	if os.Getenv("SNOOFER_DECK_BASELINE_PROBE") != "1" {
		t.Skip("manual initial-input acceptance")
	}
	paths, err := discover()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		d, err := openDevice(path)
		if err != nil {
			t.Logf("open: %v", err)
			continue
		}
		func() {
			defer d.close()
			t.Logf("query %s (%04X), input=%d", d.model.Name, d.product, d.input)
			report, err := probeInitialInput(d)
			if err != nil {
				t.Logf("query error: %v", err)
				return
			}
			decoder := Decoder{Model: d.model}
			events, decodeErr := decoder.Decode(report)
			t.Logf("report=% x; decode=%v; events=%v", report[:min(len(report), 48)], decodeErr, events)
		}()
	}
}
