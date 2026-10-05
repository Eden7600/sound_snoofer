package cli

import (
	"bytes"
	"context"
	"testing"
)

func TestMaintenanceDoesNotLaunchLegacyUI(t *testing.T) {
	for _, args := range [][]string{nil, {"tui"}, {"--dry-run"}} {
		_, deps, _, _ := setup(t)
		deps.Open = func(string) (Client, error) { t.Fatal("unexpected native initialization"); return nil, nil }
		var out bytes.Buffer
		if code := Run(context.Background(), args, &out, &out, deps); code != 2 {
			t.Fatal(args, code, out.String())
		}
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
