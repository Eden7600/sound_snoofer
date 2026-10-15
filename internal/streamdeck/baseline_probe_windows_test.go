//go:build windows

package streamdeck

import (
	"os"
	"testing"
)

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
			report, err := d.inputReport()
			if err != nil {
				t.Logf("query error: %v", err)
				return
			}
			decoder := Decoder{Model: d.model}
			events, decodeErr := decoder.Decode(report)
			t.Logf("report=% x; decode=%v; keysReady=%v encodersReady=%v events=%v", report[:min(len(report), 48)], decodeErr, decoder.keysReady, decoder.encodersReady, events)
		}()
	}
}
