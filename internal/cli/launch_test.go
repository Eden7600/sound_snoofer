package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/internal/tui"
)

func TestDefaultLaunch(t *testing.T) {
	for _, args := range [][]string{nil, {"--dry-run"}, {"tui"}, {"tui", "--dry-run"}} {
		t.Run(stringifyArgs(args), func(t *testing.T) {
			_, deps, _, _ := setup(t)
			dir := t.TempDir()
			deps.Executable = func() (string, error) { return filepath.Join(dir, "sound-snoofer.exe"), nil }
			called := false
			deps.RunTUI = func(_ context.Context, cfg config.Config, path, _ string, live bool, _ io.Writer, _ tui.Dependencies) error {
				called = true
				wantLive := len(args) == 0 || args[len(args)-1] != "--dry-run"
				if live != wantLive || path != filepath.Join(dir, "config.json") || cfg.VoiceIntent() == nil {
					t.Fatal("wrong launch", live, path)
				}
				return nil
			}
			var out, errout bytes.Buffer
			if code := Run(context.Background(), args, &out, &errout, deps); code != 0 || !called {
				t.Fatal(code, errout.String())
			}
			if _, err := os.Stat(filepath.Join(dir, "config.json")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func stringifyArgs(args []string) string {
	if len(args) == 0 {
		return "double-click"
	}
	return args[len(args)-1]
}

func TestExplicitLaunchConfig(t *testing.T) {
	_, deps, path, _ := setup(t)
	deps.Executable = func() (string, error) { t.Fatal("explicit config should not resolve executable"); return "", nil }
	deps.RunTUI = func(_ context.Context, _ config.Config, got, _ string, live bool, _ io.Writer, _ tui.Dependencies) error {
		if got != path || live {
			t.Fatal(got, live)
		}
		return nil
	}
	var out, errout bytes.Buffer
	if code := Run(context.Background(), []string{"--config", path, "--dry-run"}, &out, &errout, deps); code != 0 {
		t.Fatal(errout.String())
	}
}

func TestWatchDefaultsLive(t *testing.T) {
	client, deps, path, locks := setup(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deps.Clock = &testClock{cancel: cancel}
	var out, errout bytes.Buffer
	if code := Run(ctx, []string{"watch", "--config", path}, &out, &errout, deps); code != 0 || client.writes == 0 || *locks != 0 {
		t.Fatal("watch default or lock release", code, client.writes, *locks, errout.String())
	}
}
