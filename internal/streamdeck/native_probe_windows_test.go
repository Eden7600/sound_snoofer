//go:build windows

package streamdeck

import (
	"os"
	"testing"
)

func TestNativeHIDDiscovery(t *testing.T) {
	if os.Getenv("SNOOFER_READ_ONLY_PROBE") != "1" {
		t.Skip("manual read-only acceptance")
	}
	paths, e := discover()
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("Stream Deck interfaces: %d", len(paths))
	for _, path := range paths {
		d, e := openDevice(path)
		if e != nil {
			t.Log(e)
			continue
		}
		t.Logf("%s (%04X): report sizes input=%d output=%d feature=%d serial=%s", d.model.Name, d.product, d.input, d.output, d.feature, d.serial)
		d.close()
	}
}
