package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type failingOutput struct{}

func (failingOutput) Write([]byte) (int, error) { return 0, errors.New("sink closed") }
func TestOutputFailureStopsApplyAndWatch(t *testing.T) {
	for _, command := range []string{"devices", "plan", "apply", "watch"} {
		client, deps, p, _ := setup(t)
		args := []string{command, "--json"}
		if command != "devices" {
			args = append(args, "--config", p)
		}
		var errout bytes.Buffer
		if code := Run(context.Background(), args, failingOutput{}, &errout, deps); code != 1 || client.writes != 0 {
			t.Fatal(command, code, client.writes, errout.String())
		}
	}
}
