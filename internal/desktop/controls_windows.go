//go:build windows

package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"sound-snoofer/internal/tui"
)

type controls struct {
	updates       chan []byte
	done          chan struct{}
	cancel        context.CancelFunc
	command       *exec.Cmd
	input, output *os.File
	closeOnce     sync.Once
	exitErr       error
}

func startControls(ctx context.Context, path string, state tui.State, actions chan<- tui.Action) (*controls, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	childIn, parentOut, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	parentIn, childOut, err := os.Pipe()
	if err != nil {
		childIn.Close()
		parentOut.Close()
		return nil, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	cmd := exec.Command(exe, "__controls", path)
	cmd.Stdin = childIn
	cmd.Stdout = childOut
	cmd.Stderr = io.Discard
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	c := &controls{updates: make(chan []byte, 1), done: make(chan struct{}), cancel: cancel, command: cmd, input: parentIn, output: parentOut}
	err = cmd.Start()
	childIn.Close()
	childOut.Close()
	if err != nil {
		c.close()
		return nil, err
	}
	// Pipe closure unblocks both I/O goroutines, including a stalled renderer.
	var ioDone sync.WaitGroup
	ioDone.Add(2)
	go func() {
		defer ioDone.Done()
		// EOF and cancellation detach this session; they do not stop the actor.
		_ = readActions(runCtx, parentIn, actions)
		c.close()
	}()
	go func() {
		defer ioDone.Done()
		_ = writeFrames(runCtx, parentOut, c.updates)
		c.close()
	}()
	go func() {
		c.exitErr = cmd.Wait()
		c.close()
		ioDone.Wait()
		close(c.done)
	}()
	if err := c.publish(state); err != nil {
		c.stop()
		return nil, err
	}
	return c, nil
}

func (c *controls) close() {
	c.closeOnce.Do(func() {
		c.cancel()
		// These handles may already be broken after child exit; closing is best effort.
		_ = c.input.Close()
		_ = c.output.Close()
	})
}
func (c *controls) stop() {
	c.close()
	select {
	case <-c.done:
	case <-time.After(2 * time.Second):
		// Only this tray's controls child is eligible for forced shutdown.
		_ = c.command.Process.Kill()
	}
}
func (c *controls) publish(state tui.State) error {
	data, err := json.Marshal(frame{State: &state})
	if err != nil {
		return fmt.Errorf("encode controls state: %w", err)
	}
	offer(c.updates, data)
	return nil
}
func (c *controls) focus(state tui.State) {
	data, err := json.Marshal(frame{State: &state, Focus: true})
	if err == nil {
		offer(c.updates, data)
	}
}

// RunControls is the private attached renderer. It never opens the audio DLL.
func RunControls(ctx context.Context, path string) error {
	pipeIn, pipeOut := os.Stdin, os.Stdout
	in, out, err := controlsConsole()
	if err != nil {
		return err
	}
	configureControlsFont(out)
	setControlsIcon()
	focusConsole()
	defer in.Close()
	defer out.Close()
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer pipeIn.Close()
	defer pipeOut.Close()
	states := make(chan tui.State, 1)
	actions := make(chan tui.Action, 8)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		defer close(states)
		defer cancel()
		decoder := json.NewDecoder(pipeIn)
		for {
			var f frame
			if err := decoder.Decode(&f); err != nil {
				return
			}
			if f.Focus {
				focusConsole()
			}
			if f.State != nil {
				select {
				case states <- *f.State:
				case <-runCtx.Done():
					return
				}
			}
		}
	}()
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		encoder := json.NewEncoder(pipeOut)
		for {
			select {
			case <-runCtx.Done():
				return
			case action := <-actions:
				if err := encoder.Encode(action); err != nil {
					cancel()
					return
				}
			}
		}
	}()
	select {
	case initial, ok := <-states:
		if ok {
			err = tui.RunConnected(runCtx, path, initial, actions, states, in, out, true)
		}
	case <-runCtx.Done():
	}
	cancel()
	pipeIn.Close()
	pipeOut.Close()
	<-readerDone
	<-writerDone
	return err
}
