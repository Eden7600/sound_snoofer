//go:build windows

package vrclipboard

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/windows"

	"sound-snoofer/snoofer"
)

const maxWorkerStatus = 4096

type workerStatus struct {
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

func (s workerStatus) valid() bool {
	return (s.State == "ready" || s.State == "waiting" || s.State == "error") && utf8.ValidString(s.Detail) && utf8.RuneCountInString(s.Detail) <= 512
}

func writeWorkerStatus(output io.Writer, status workerStatus) error {
	status.Detail = strings.ToValidUTF8(status.Detail, "�")
	if runes := []rune(status.Detail); len(runes) > 512 {
		status.Detail = string(runes[:512])
	}
	if !status.valid() {
		return fmt.Errorf("invalid worker state")
	}
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	n, err := output.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return err
}

func readWorkerStatuses(ctx context.Context, input io.Reader, updates chan<- workerStatus) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 1024), maxWorkerStatus)
	for scanner.Scan() {
		var status workerStatus
		decoder := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&status); err != nil || !status.valid() {
			return fmt.Errorf("invalid worker status")
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			return fmt.Errorf("invalid worker status trailing data")
		}
		select {
		case updates <- status:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("worker status pipe failed: %w", err)
	}
	return nil
}

type workerChild interface {
	Updates() <-chan workerStatus
	Done() <-chan struct{}
	Err() error // Called after Done closes.
	Stop(context.Context) error
}

type supervisorOptions struct {
	running func() (bool, error)
	start   func(string) (workerChild, error)
	ticks   func() (<-chan time.Time, func())
	grace   time.Duration
}

func superviseWorker(ctx context.Context, s snoofer.Services, options supervisorOptions) (result error) {
	var link snoofer.ConnectionTracker
	var child workerChild
	var updates <-chan workerStatus
	var done <-chan struct{}
	status := workerStatus{State: "waiting", Detail: "SteamVR is not running"}
	var detectionErr error
	blocked := false
	publish := func() {
		now := time.Now()
		state, value, detail := snoofer.ConnectionDisconnected, "Waiting", status.Detail
		switch {
		case !s.Live:
			state, value, detail = snoofer.ConnectionOff, "Preview", "No overlay worker in preview"
		case detectionErr != nil:
			state, value, detail = snoofer.ConnectionUnknown, "Unknown", detectionErr.Error()
		case status.State == "error":
			state, value = snoofer.ConnectionError, "Error"
		case status.State == "ready":
			state, value = snoofer.ConnectionConnected, "Ready"
			link.Activity(now)
		case child != nil:
			state, value = snoofer.ConnectionConnecting, "Connecting"
		}
		if state == snoofer.ConnectionError || detectionErr != nil {
			link.Fail(detail, now)
		}
		link.Observe(state, now)
		// Metadata is fixed; publication cannot conflict with another provider.
		_ = s.Controls.Publish("vrclipboard", []snoofer.Control{
			{ID: "vrclipboard.status", Label: "VR Clipboard", Group: "VR Clipboard", Kind: "status", Value: value, Status: detail, Available: true},
			{ID: "vrclipboard.app-steamvr", Label: "SteamVR Clipboard", Group: "VR Clipboard", Kind: "connection", Value: value, Status: detail, SurfaceOnly: true, Available: true,
				Connection: link.Report("vrserver.exe (this session)", snoofer.ConnectionDetail{Label: "Interval", Value: "1s"})},
		}, nil)
	}
	stop := func() error {
		if child == nil {
			return nil
		}
		stopCtx, cancel := context.WithTimeout(context.Background(), options.grace)
		defer cancel()
		updates, done = nil, nil
		err := child.Stop(stopCtx)
		if err == nil {
			child = nil
		}
		return err
	}
	defer func() { result = errors.Join(result, stop()) }()
	if !s.Live {
		publish()
		<-ctx.Done()
		return nil
	}
	ticks, stopTicks := options.ticks()
	defer stopTicks()
	observe := func() {
		if ctx.Err() != nil {
			return
		}
		var running bool
		running, detectionErr = options.running()
		if detectionErr == nil && !running {
			blocked = false
			status = workerStatus{State: "waiting", Detail: "SteamVR is not running"}
			if err := stop(); err != nil {
				status = workerStatus{State: "error", Detail: err.Error()}
			}
		} else if detectionErr == nil && child == nil && !blocked && ctx.Err() == nil {
			blocked = true
			var err error
			child, err = options.start(s.Path)
			status = workerStatus{State: "waiting", Detail: "Starting overlay worker"}
			if err != nil {
				child = nil
				status = workerStatus{State: "error", Detail: err.Error()}
			} else {
				updates, done = child.Updates(), child.Done()
			}
		}
		publish()
	}
	observe()
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-ticks:
			if !ok {
				return nil
			}
			observe()
		case next, ok := <-updates:
			if ok {
				status = next
			} else {
				updates = nil
				ended := child
				if err := stop(); err != nil {
					status = workerStatus{State: "error", Detail: err.Error()}
				} else if err := ended.Err(); err != nil && status.State != "error" {
					status = workerStatus{State: "error", Detail: "Overlay worker exited: " + err.Error()}
				} else if status.State != "error" {
					status = workerStatus{State: "waiting", Detail: "Worker stopped; restart SteamVR to reconnect"}
				}
			}
			publish()
		case <-done:
			// Exit may race the last status; retain a buffered startup failure.
			for next := range updates {
				status = next
			}
			if err := child.Err(); err != nil && status.State != "error" {
				status = workerStatus{State: "error", Detail: "Overlay worker exited: " + err.Error()}
			} else if status.State != "error" {
				status = workerStatus{State: "waiting", Detail: "Worker stopped; restart SteamVR to reconnect"}
			}
			if err := stop(); err != nil {
				status = workerStatus{State: "error", Detail: err.Error()}
			}
			publish()
		}
	}
}

type workerProcess struct {
	input, output *os.File
	job           windows.Handle
	updates       chan workerStatus
	done          chan struct{}
	err           error
	closeInput    sync.Once
	closeJob      sync.Once
	jobErr        error
}

func (p *workerProcess) Updates() <-chan workerStatus { return p.updates }
func (p *workerProcess) Done() <-chan struct{}        { return p.done }
func (p *workerProcess) Err() error                   { return p.err }

func (p *workerProcess) releaseJob() error {
	p.closeJob.Do(func() { p.jobErr = windows.CloseHandle(p.job) })
	return p.jobErr
}

func (p *workerProcess) Stop(ctx context.Context) error {
	p.closeInput.Do(func() { _ = p.input.Close() })
	select {
	case <-p.done:
		return p.releaseJob()
	case <-ctx.Done():
		// Closing this non-inherited job can terminate only our owned worker.
		err := p.releaseJob()
		_ = p.output.Close()
		select {
		case <-p.done:
			return err
		case <-time.After(500 * time.Millisecond):
			return errors.Join(err, fmt.Errorf("overlay worker did not exit after shutdown"))
		}
	}
}

func workerCommand(executable, config string) *exec.Cmd {
	cmd := exec.Command(executable, "--config", config, "vrclipboard", "worker")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP, HideWindow: true}
	return cmd
}

func startWorkerProcess(config string) (workerChild, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("create overlay worker job: %w", err)
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	// Runtime descendants must not be terminated if initialization starts SteamVR.
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK
	if _, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		_ = windows.CloseHandle(job)
		return nil, fmt.Errorf("configure overlay worker job: %w", err)
	}
	var files []*os.File
	cleanup := func() {
		for _, file := range files {
			_ = file.Close()
		}
		_ = windows.CloseHandle(job)
	}
	childIn, parentOut, err := os.Pipe()
	if err != nil {
		cleanup()
		return nil, err
	}
	files = append(files, childIn, parentOut)
	parentIn, childOut, err := os.Pipe()
	if err != nil {
		cleanup()
		return nil, err
	}
	files = append(files, parentIn, childOut)
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		cleanup()
		return nil, err
	}
	files = append(files, null)
	cmd := workerCommand(executable, config)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = childIn, childOut, null
	if err = cmd.Start(); err != nil {
		cleanup()
		return nil, fmt.Errorf("start overlay worker: %w", err)
	}
	_ = childIn.Close()
	_ = childOut.Close()
	_ = null.Close()
	handle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err == nil {
		err = windows.AssignProcessToJobObject(job, handle)
		_ = windows.CloseHandle(handle)
	}
	if err == nil {
		_, err = parentOut.Write([]byte{1})
	}
	if err != nil {
		// The startup gate has not opened; kill only the child we just created.
		killErr := cmd.Process.Kill()
		cleanup()
		waitDone := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(waitDone)
		}()
		select {
		case <-waitDone:
		case <-time.After(500 * time.Millisecond):
			err = errors.Join(err, killErr, fmt.Errorf("overlay worker startup cleanup did not finish"))
		}
		return nil, fmt.Errorf("attach overlay worker: %w", err)
	}
	p := &workerProcess{input: parentOut, output: parentIn, job: job, updates: make(chan workerStatus, 4), done: make(chan struct{})}
	readCtx, cancelRead := context.WithCancel(context.Background())
	readDone := make(chan error, 1)
	go func() {
		defer close(p.updates)
		readDone <- readWorkerStatuses(readCtx, parentIn, p.updates)
	}()
	go func() {
		defer close(p.done)
		p.err = cmd.Wait()
		// Give the reader a bounded chance to retain the final status before EOF;
		// a runtime descendant holding inherited pipes must not delay shutdown.
		var readErr error
		select {
		case readErr = <-readDone:
		case <-time.After(100 * time.Millisecond):
			cancelRead()
			_ = parentIn.Close()
			readErr = <-readDone
		}
		cancelRead()
		_ = parentIn.Close()
		if readErr != nil && !errors.Is(readErr, context.Canceled) && !errors.Is(readErr, os.ErrClosed) {
			p.err = errors.Join(p.err, readErr)
		}
	}()
	return p, nil
}
