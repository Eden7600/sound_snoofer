//go:build windows

package app

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

	"golang.org/x/sys/windows"
)

type controls struct {
	updates       chan []byte
	done          chan struct{}
	cancel        context.CancelFunc
	command       *exec.Cmd
	input, output *os.File
	closeOnce     sync.Once
	exitErr       error
	animations    animationSender // Used only by the tray goroutine that publishes.
}

func startControls(ctx context.Context, state ViewState, actions chan<- UIAction) (*controls, error) {
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
	cmd := exec.Command(exe, "__controls")
	cmd.Stdin = childIn
	cmd.Stdout = childOut
	cmd.Stderr = io.Discard
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS}
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
func (c *controls) publish(state ViewState) error {
	data, err := json.Marshal(frame{State: &state, Animations: c.animations.changed(state.Controls)})
	if err != nil {
		return fmt.Errorf("encode controls state: %w", err)
	}
	offer(c.updates, data)
	return nil
}
func (c *controls) focus(state ViewState) {
	data, err := json.Marshal(frame{State: &state, Focus: true})
	if err == nil {
		offer(c.updates, data)
	}
}
