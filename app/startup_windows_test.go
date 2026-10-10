//go:build windows

package app

import (
	"errors"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// scratchStartup uses keys under a test-only HKCU path, never the real Run key.
func scratchStartup(t *testing.T) startupEntry {
	t.Helper()
	base := `Software\Snoofer-test\` + t.Name()
	entry := startupEntry{runPath: base + `\Run`, approvedPath: base + `\Approved`, name: "Snoofer"}
	t.Cleanup(func() {
		for _, path := range []string{entry.runPath, entry.approvedPath, base, `Software\Snoofer-test`} {
			if err := registry.DeleteKey(registry.CURRENT_USER, path); err != nil && !errors.Is(err, registry.ErrNotExist) {
				t.Logf("delete %s: %v", path, err)
			}
		}
	})
	return entry
}

func setApproval(t *testing.T, entry startupEntry, first byte) {
	t.Helper()
	key, _, err := registry.CreateKey(registry.CURRENT_USER, entry.approvedPath, registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer key.Close()
	data := make([]byte, 12)
	data[0] = first
	if err := key.SetBinaryValue(entry.name, data); err != nil {
		t.Fatal(err)
	}
}

func TestStartupEntryRoundTrip(t *testing.T) {
	entry := scratchStartup(t)
	const command = `"C:\bin\snoofer.exe" --config "C:\bin\snoofer.json"`

	if got, err := entry.read(); err != nil || got != (startupState{}) {
		t.Fatalf("absent read = %+v, %v", got, err)
	}
	if err := entry.apply(false, command); err != nil {
		t.Fatalf("removing an absent entry: %v", err)
	}
	if err := entry.apply(true, command); err != nil {
		t.Fatal(err)
	}
	if got, err := entry.read(); err != nil || got != (startupState{Command: command}) {
		t.Fatalf("registered read = %+v, %v", got, err)
	}

	setApproval(t, entry, 0x03)
	if got, err := entry.read(); err != nil || !got.Disabled {
		t.Fatalf("disabled read = %+v, %v", got, err)
	}
	setApproval(t, entry, 0x02)
	if got, err := entry.read(); err != nil || got.Disabled {
		t.Fatalf("enabled approval read = %+v, %v", got, err)
	}

	setApproval(t, entry, 0x03)
	if err := entry.apply(true, command); err != nil {
		t.Fatal(err)
	}
	if got, err := entry.read(); err != nil || got != (startupState{Command: command}) {
		t.Fatalf("turning on did not clear the Windows disable: %+v, %v", got, err)
	}

	if err := entry.apply(false, command); err != nil {
		t.Fatal(err)
	}
	if got, err := entry.read(); err != nil || got != (startupState{}) {
		t.Fatalf("removed read = %+v, %v", got, err)
	}
}

func TestStartupEntryReplacesAnotherCopy(t *testing.T) {
	entry := scratchStartup(t)
	if err := entry.apply(true, `"D:\old\snoofer.exe" --config "D:\old\snoofer.json"`); err != nil {
		t.Fatal(err)
	}
	const command = `"C:\bin\snoofer.exe" --config "C:\bin\snoofer.json"`
	if err := entry.apply(true, command); err != nil {
		t.Fatal(err)
	}
	if got, err := entry.read(); err != nil || got.Command != command {
		t.Fatalf("read = %+v, %v", got, err)
	}
}
