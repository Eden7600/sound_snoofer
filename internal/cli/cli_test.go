package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"voice-snooter/internal/model"
)

type testClient struct {
	writes, closes int
	err            error
	s              model.Snapshot
}

func (c *testClient) Snapshot() (model.Snapshot, error) { return c.s, c.err }
func (c *testClient) Set(target string, d model.Device) error {
	c.writes++
	c.s.Assignments[target] = d.Name
	return nil
}
func (c *testClient) Close() error { c.closes++; return nil }

type testClock struct {
	now    time.Time
	waits  int
	cancel context.CancelFunc
}

func (c *testClock) Now() time.Time { return c.now }
func (c *testClock) Wait(ctx context.Context, d time.Duration) error {
	c.now = c.now.Add(d)
	c.waits++
	if c.waits >= 4 && c.cancel != nil {
		c.cancel()
	}
	return ctx.Err()
}
func setup(t *testing.T) (*testClient, Deps, string, *int) {
	t.Helper()
	client := &testClient{s: model.Snapshot{Edition: 2, Assignments: map[string]string{"A1": "old"}, Devices: []model.Device{{Name: "speakers", Driver: "wdm", Direction: "output", Available: true}}}}
	locks := new(int)
	d := Deps{Open: func(string) (Client, error) { return client, nil }, Clock: &testClock{}, Acquire: func() (func(), error) { *locks++; return func() { *locks-- }, nil }}
	path := filepath.Join(t.TempDir(), "config.json")
	if e := os.WriteFile(path, []byte(`{"version":1,"routes":[{"target":"A1","candidates":[{"driver":"wdm","pattern":"speakers"}]}]}`), 0600); e != nil {
		t.Fatal(e)
	}
	return client, d, path, locks
}
func TestReadOnlyCommands(t *testing.T) {
	for _, cmd := range []string{"devices", "plan", "watch"} {
		t.Run(cmd, func(t *testing.T) {
			client, d, path, locks := setup(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d.Clock = &testClock{cancel: cancel}
			args := []string{cmd, "--json"}
			if cmd != "devices" {
				args = append(args, "--config", path)
			}
			var out, errout bytes.Buffer
			code := Run(ctx, args, &out, &errout, d)
			if code != 0 || client.writes != 0 || client.closes != 1 || *locks != 0 {
				t.Fatal(code, client, errout.String())
			}
			if !strings.Contains(out.String(), "speakers") {
				t.Fatal(out.String())
			}
		})
	}
}
func TestApplyAndRelease(t *testing.T) {
	client, d, path, locks := setup(t)
	var out, errout bytes.Buffer
	if code := Run(context.Background(), []string{"apply", "--config", path}, &out, &errout, d); code != 0 || client.writes != 1 || client.closes != 1 || *locks != 0 {
		t.Fatal(code, errout.String(), client, *locks)
	}
}
func TestCLIValidationBeforeOpen(t *testing.T) {
	for _, args := range [][]string{{"plan"}, {"bogus"}, {"devices", "--apply"}, {"watch", "--config", "missing"}, {"devices", "extra"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, d, _, _ := setup(t)
			opened := false
			d.Open = func(string) (Client, error) { opened = true; return nil, errors.New("unexpected") }
			var out, errout bytes.Buffer
			if code := Run(context.Background(), args, &out, &errout, d); code != 2 || opened {
				t.Fatal(code, opened)
			}
		})
	}
}
func TestConnectionErrorAndWriterConflict(t *testing.T) {
	client, d, path, _ := setup(t)
	client.err = errors.New("Voicemeeter disconnected")
	var out, errout bytes.Buffer
	if code := Run(context.Background(), []string{"devices"}, &out, &errout, d); code != 1 || !strings.Contains(errout.String(), "disconnected") {
		t.Fatal(code, errout.String())
	}
	client, d, path, _ = setup(t)
	d.Acquire = func() (func(), error) { return nil, errors.New("writer active") }
	errout.Reset()
	if code := Run(context.Background(), []string{"watch", "--config", path, "--apply"}, &out, &errout, d); code != 1 || client.writes != 0 || client.closes != 0 {
		t.Fatal(code, errout.String())
	}
}
