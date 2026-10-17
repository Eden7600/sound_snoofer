//go:build windows

package vroverlay

import (
	"errors"
	"strings"
	"testing"
)

func TestRuntimePathUsesOnlyActiveAbsoluteEntry(t *testing.T) {
	got, err := runtimePath(strings.NewReader(`{"runtime":["D:\\Steam\\SteamVR","E:\\old"],"config":["unused"]}`))
	if err != nil || got != `D:\Steam\SteamVR` {
		t.Fatal(got, err)
	}
	for _, data := range []string{`{`, `{}`, `{"runtime":[]}`, `{"runtime":["relative","D:\\valid-history"]}`, `{"runtime":["C:relative"]}`, `{"runtime":["C:\\bad\u0000"]}`, `{"runtime":"C:\\Steam"}`, strings.Repeat(" ", maxPathRegistry+1)} {
		if _, err := runtimePath(strings.NewReader(data)); err == nil {
			t.Fatal("accepted invalid runtime registry")
		}
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestRuntimePathReadFailure(t *testing.T) {
	if _, err := runtimePath(failingReader{}); err == nil {
		t.Fatal("read error ignored")
	}
}
