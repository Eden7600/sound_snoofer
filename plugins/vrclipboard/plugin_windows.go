//go:build windows

// Package vrclipboard provides an isolated SteamVR clipboard pad.
package vrclipboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"sound-snoofer/internal/process"
	"sound-snoofer/snoofer"
)

type instance struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

func (i *instance) Stop(ctx context.Context) error {
	if err := snoofer.CancelAndWait(ctx, i.cancel, i.done); err != nil {
		return err
	}
	return i.err
}

// Plugin returns inert metadata; preview never observes processes or launches a worker.
func Plugin() snoofer.Plugin {
	return pluginWithSupervisor(supervisorOptions{
		running: func() (bool, error) {
			names, err := process.Names()
			return process.Contains(names, "vrserver.exe"), err
		},
		start: startWorkerProcess,
		ticks: func() (<-chan time.Time, func()) {
			ticker := time.NewTicker(time.Second)
			return ticker.C, ticker.Stop
		},
		grace: 2 * time.Second,
	})
}

func pluginWithSupervisor(options supervisorOptions) snoofer.Plugin {
	validate := func(raw json.RawMessage) error { return snoofer.DecodeSettings(raw, &struct{}{}) }
	return snoofer.Plugin{
		ID: "vrclipboard", Label: "VR Clipboard", Defaults: json.RawMessage(`{}`), Validate: validate,
		Command: func(ctx context.Context, s snoofer.Services, _ json.RawMessage, args []string) error {
			return runWorkerCommand(ctx, s.Live, args, os.Stdin, os.Stdout, RunWorker)
		},
		Start: func(ctx context.Context, s snoofer.Services, raw json.RawMessage, _ map[string]snoofer.Instance) (snoofer.Instance, error) {
			if err := validate(raw); err != nil {
				return nil, err
			}
			runCtx, cancel := context.WithCancel(ctx)
			i := &instance{cancel: cancel, done: make(chan struct{})}
			go func() {
				defer close(i.done)
				defer s.Controls.Remove("vrclipboard")
				i.err = superviseWorker(runCtx, s, options)
			}()
			return i, nil
		},
	}
}

func runWorkerCommand(ctx context.Context, live bool, args []string, input io.Reader, output io.Writer, run func(context.Context, io.Reader, io.Writer) error) error {
	if len(args) != 1 || args[0] != "worker" {
		return fmt.Errorf("expected vrclipboard worker")
	}
	if !live {
		// Command errors normally open the host's fatal dialog. This hidden command
		// reports through its pipe and exits normally, including a broken pipe.
		_ = writeWorkerStatus(output, workerStatus{State: "waiting", Detail: "Preview"})
		return nil
	}
	if ctx.Err() != nil {
		return nil
	}
	if closer, ok := input.(io.Closer); ok {
		stopClose := context.AfterFunc(ctx, func() { _ = closer.Close() })
		defer stopClose()
	}
	var token [1]byte
	if _, err := io.ReadFull(input, token[:]); err != nil || token[0] != 1 || ctx.Err() != nil {
		return nil
	}
	if err := run(ctx, input, output); err != nil {
		_ = writeWorkerStatus(output, workerStatus{State: "error", Detail: err.Error()})
	}
	return nil
}
