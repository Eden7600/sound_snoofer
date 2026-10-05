package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIConfigAdjacentMuteJournal(t *testing.T) {
	for _, command := range []string{"apply", "watch"} {
		for _, override := range []bool{false, true} {
			client, deps, p, _ := setup(t)
			if e := os.WriteFile(p+".mutes.json", []byte("invalid"), 0600); e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithCancel(context.Background())
			deps.Clock = &testClock{cancel: cancel}
			defer cancel()
			args := []string{command, "--config", p}
			if override {
				args = append(args, "--dll", filepath.Join(t.TempDir(), "remote.dll"))
			}
			var out bytes.Buffer
			code := Run(ctx, args, &out, &out, deps)
			if !strings.Contains(out.String(), "invalid") || client.writes != 0 {
				t.Fatal(command, override, code, out.String(), client.writes)
			}
		}
	}
}
