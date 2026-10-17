//go:build windows

package vrclipboard

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"golang.org/x/sys/windows"

	"sound-snoofer/snoofer"
)

type fakeWorkerChild struct {
	updates  chan workerStatus
	done     chan struct{}
	once     sync.Once
	err      error
	stopped  atomic.Int32
	failStop atomic.Bool
}

func newFakeWorker() *fakeWorkerChild {
	return &fakeWorkerChild{updates: make(chan workerStatus, 8), done: make(chan struct{})}
}
func (f *fakeWorkerChild) Updates() <-chan workerStatus { return f.updates }
func (f *fakeWorkerChild) Done() <-chan struct{}        { return f.done }
func (f *fakeWorkerChild) Err() error                   { return f.err }
func (f *fakeWorkerChild) finish(err error) {
	f.once.Do(func() {
		f.err = err
		close(f.updates)
		close(f.done)
	})
}
func (f *fakeWorkerChild) Stop(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("shutdown missing deadline")
	}
	f.stopped.Add(1)
	if f.failStop.Load() {
		return errors.New("worker cleanup unconfirmed")
	}
	f.finish(nil)
	return nil
}

func waitSupervisor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("supervisor did not reach expected state")
		}
		time.Sleep(time.Millisecond)
	}
}

func supervisorControl(registry *snoofer.Controls) snoofer.Control {
	for _, control := range registry.Snapshot() {
		if control.ID == "vrclipboard.status" {
			return control
		}
	}
	return snoofer.Control{}
}

func TestSupervisorPreviewNeverTouchesRuntimeOrProcess(t *testing.T) {
	unexpected := func() { t.Error("preview called native dependency") }
	plugin := pluginWithSupervisor(supervisorOptions{
		running: func() (bool, error) { unexpected(); return true, nil },
		start:   func(string) (workerChild, error) { unexpected(); return nil, nil },
		ticks:   func() (<-chan time.Time, func()) { unexpected(); return nil, func() {} },
	})
	if len(plugin.Requires) != 0 || snoofer.Defaults([]snoofer.Plugin{plugin}).Plugins["vrclipboard"].Enabled {
		t.Fatal("overlay gained a dependency or enabled default")
	}
	registry := snoofer.NewControls()
	instance, err := plugin.Start(context.Background(), snoofer.Services{Controls: registry}, []byte(`{}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	waitSupervisor(t, func() bool { return supervisorControl(registry).Value == "Preview" })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := instance.Stop(ctx); err != nil || len(registry.Snapshot()) != 0 {
		t.Fatal("preview cleanup", err, registry.Snapshot())
	}
	if err := plugin.Validate([]byte(`{"enabled":true}`)); err == nil {
		t.Fatal("unexpected settings accepted")
	}
}

type supervisorFixture struct {
	registry *snoofer.Controls
	ticks    chan time.Time
	observed atomic.Int32
	starts   atomic.Int32
	mu       sync.Mutex
	running  bool
	detect   error
	startErr error
	children []*fakeWorkerChild
	instance snoofer.Instance
}

func newSupervisorFixture(t *testing.T) *supervisorFixture {
	t.Helper()
	f := &supervisorFixture{registry: snoofer.NewControls(), ticks: make(chan time.Time, 1)}
	options := supervisorOptions{
		running: func() (bool, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.observed.Add(1)
			return f.running, f.detect
		},
		start: func(path string) (workerChild, error) {
			if path != `C:\exact path\personal.json` {
				t.Errorf("changed configuration path: %q", path)
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			f.starts.Add(1)
			if f.startErr != nil {
				return nil, f.startErr
			}
			child := newFakeWorker()
			f.children = append(f.children, child)
			return child, nil
		},
		ticks: func() (<-chan time.Time, func()) { return f.ticks, func() {} },
		grace: 50 * time.Millisecond,
	}
	var err error
	f.instance, err = pluginWithSupervisor(options).Start(context.Background(), snoofer.Services{Controls: f.registry, Live: true, Path: `C:\exact path\personal.json`}, []byte(`{}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	waitSupervisor(t, func() bool { return supervisorControl(f.registry).Value == "Waiting" })
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := f.instance.Stop(ctx); err != nil {
			t.Error(err)
		}
	})
	return f
}

func (f *supervisorFixture) observe(t *testing.T, running bool, err error) {
	t.Helper()
	f.mu.Lock()
	f.running, f.detect = running, err
	f.mu.Unlock()
	before := f.observed.Load()
	f.ticks <- time.Now()
	waitSupervisor(t, func() bool { return f.observed.Load() > before })
}

func (f *supervisorFixture) child(t *testing.T, index int) *fakeWorkerChild {
	t.Helper()
	var child *fakeWorkerChild
	waitSupervisor(t, func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		if len(f.children) > index {
			child = f.children[index]
		}
		return child != nil
	})
	return child
}

func TestSupervisorRequiresRuntimeAndSuppressesRestartAfterExit(t *testing.T) {
	f := newSupervisorFixture(t)
	f.observe(t, false, errors.New("process snapshot unavailable"))
	waitSupervisor(t, func() bool { return supervisorControl(f.registry).Value == "Unknown" })
	if f.starts.Load() != 0 {
		t.Fatal("unknown runtime launched child")
	}
	f.observe(t, true, nil)
	first := f.child(t, 0)
	first.updates <- workerStatus{State: "ready"}
	waitSupervisor(t, func() bool { return supervisorControl(f.registry).Value == "Ready" })
	first.finish(nil)
	waitSupervisor(t, func() bool { return strings.Contains(supervisorControl(f.registry).Status, "restart SteamVR") })
	for range 3 {
		f.observe(t, true, nil)
	}
	if f.starts.Load() != 1 || first.stopped.Load() != 1 {
		t.Fatal("worker restarted within runtime lifetime", f.starts.Load(), first.stopped.Load())
	}
	f.observe(t, false, nil)
	waitSupervisor(t, func() bool { return supervisorControl(f.registry).Status == "SteamVR is not running" })
	f.observe(t, true, nil)
	_ = f.child(t, 1)
}

func TestSupervisorRuntimeAbsenceStopsChildButDetectionErrorDoesNot(t *testing.T) {
	f := newSupervisorFixture(t)
	f.observe(t, true, nil)
	child := f.child(t, 0)
	child.updates <- workerStatus{State: "ready"}
	waitSupervisor(t, func() bool { return supervisorControl(f.registry).Value == "Ready" })
	f.observe(t, false, errors.New("snapshot failed"))
	waitSupervisor(t, func() bool { return supervisorControl(f.registry).Value == "Unknown" })
	if child.stopped.Load() != 0 {
		t.Fatal("uncertain runtime state stopped worker")
	}
	f.observe(t, false, nil)
	waitSupervisor(t, func() bool { return child.stopped.Load() == 1 })
	f.observe(t, true, nil)
	_ = f.child(t, 1)
}

func TestSupervisorRetainsFinalWorkerErrorAndStartupFailure(t *testing.T) {
	f := newSupervisorFixture(t)
	f.mu.Lock()
	f.startErr = errors.New("job setup failed")
	f.mu.Unlock()
	f.observe(t, true, nil)
	waitSupervisor(t, func() bool { return supervisorControl(f.registry).Status == "job setup failed" })
	f.observe(t, true, nil)
	if f.starts.Load() != 1 {
		t.Fatal("startup failure retried automatically")
	}
	f.observe(t, false, nil)
	f.mu.Lock()
	f.startErr = nil
	f.mu.Unlock()
	f.observe(t, true, nil)
	child := f.child(t, 0)
	child.updates <- workerStatus{State: "error", Detail: "overlay interface unavailable"}
	child.finish(errors.New("exit status 1"))
	waitSupervisor(t, func() bool { return child.stopped.Load() == 1 })
	waitSupervisor(t, func() bool { return supervisorControl(f.registry).Status == "overlay interface unavailable" })
}

func TestSupervisorClosedStatusPipeStopsWorker(t *testing.T) {
	f := newSupervisorFixture(t)
	f.observe(t, true, nil)
	child := f.child(t, 0)
	child.finish(errors.New("invalid worker status"))
	waitSupervisor(t, func() bool { return supervisorControl(f.registry).Value == "Error" })
	f.observe(t, true, nil)
	if f.starts.Load() != 1 {
		t.Fatal("pipe failure immediately restarted worker")
	}
}

func TestSupervisorDoesNotReplaceWorkerAfterUnconfirmedCleanup(t *testing.T) {
	f := newSupervisorFixture(t)
	f.observe(t, true, nil)
	child := f.child(t, 0)
	child.failStop.Store(true)
	f.observe(t, false, nil)
	waitSupervisor(t, func() bool { return supervisorControl(f.registry).Status == "worker cleanup unconfirmed" })
	f.observe(t, true, nil)
	if f.starts.Load() != 1 {
		t.Fatal("unjoined worker was replaced")
	}
	child.failStop.Store(false)
	f.observe(t, false, nil)
	waitSupervisor(t, func() bool { return child.stopped.Load() == 2 })
	f.observe(t, true, nil)
	_ = f.child(t, 1)
}

func TestWorkerStatusProtocolIsBounded(t *testing.T) {
	var output bytes.Buffer
	if err := writeWorkerStatus(&output, workerStatus{State: "error", Detail: strings.Repeat("界", 600)}); err != nil {
		t.Fatal(err)
	}
	updates := make(chan workerStatus, 4)
	if err := readWorkerStatuses(context.Background(), &output, updates); err != nil {
		t.Fatal(err)
	}
	if status := <-updates; status.State != "error" || utf8.RuneCountInString(status.Detail) != 512 {
		t.Fatal("status was not bounded", status)
	}
	for _, raw := range []string{
		`{"state":"other"}`, `{"state":"ready","title":"private"}`, `{"state":"ready"} {}`,
		`{"state":"ready","detail":"` + strings.Repeat("x", 513) + `"}`,
		strings.Repeat("x", maxWorkerStatus+1), `null`, `not json`,
	} {
		if err := readWorkerStatuses(context.Background(), strings.NewReader(raw+"\n"), updates); err == nil {
			t.Errorf("invalid status accepted: %.70s", raw)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := readWorkerStatuses(ctx, strings.NewReader(`{"state":"ready"}`+"\n"), make(chan workerStatus)); !errors.Is(err, context.Canceled) {
		t.Fatal("blocked status delivery ignored cancellation", err)
	}
	if err := writeWorkerStatus(shortStatusWriter{}, workerStatus{State: "ready"}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal("partial output accepted", err)
	}
}

type shortStatusWriter struct{}

func (shortStatusWriter) Write([]byte) (int, error) { return 0, nil }

func TestWorkerCommandWaitsForOwnedStartupAndNeverReturnsWorkerError(t *testing.T) {
	for _, test := range []struct {
		name  string
		live  bool
		input string
		calls int
	}{
		{"preview", false, "\x01", 0}, {"parent lost", true, "", 0},
		{"wrong token", true, "x", 0}, {"owned", true, "\x01", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			calls := 0
			err := runWorkerCommand(context.Background(), test.live, []string{"worker"}, strings.NewReader(test.input), &output,
				func(context.Context, io.Reader, io.Writer) error { calls++; return errors.New("native init failed") })
			if err != nil || calls != test.calls {
				t.Fatal("worker escaped owned startup/error boundary", err, calls)
			}
			if calls == 1 && !strings.Contains(output.String(), "native init failed") {
				t.Fatal("worker failure was not reported")
			}
		})
	}
	if err := runWorkerCommand(context.Background(), true, []string{"worker", "extra"}, nil, nil, nil); err == nil {
		t.Fatal("unexpected worker arguments accepted")
	}
}

func TestWorkerCommandCancellationClosesStartupPipe(t *testing.T) {
	input, writer := io.Pipe()
	defer writer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runWorkerCommand(ctx, true, []string{"worker"}, input, io.Discard,
			func(context.Context, io.Reader, io.Writer) error { t.Error("cancelled startup ran worker"); return nil })
	}()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("startup read ignored cancellation")
	}
	_ = input.Close()
}

func TestWorkerCommandUsesExactExecutableAndConfigurationWithoutLaunching(t *testing.T) {
	cmd := workerCommand(`C:\Snoofer\snoofer.exe`, `D:\User Files\snoofer.json`)
	want := []string{`C:\Snoofer\snoofer.exe`, "--config", `D:\User Files\snoofer.json`, "vrclipboard", "worker"}
	if !reflect.DeepEqual(cmd.Args, want) || cmd.Path != want[0] || !cmd.SysProcAttr.HideWindow || cmd.SysProcAttr.CreationFlags&windows.DETACHED_PROCESS == 0 {
		t.Fatalf("incorrect hidden worker command: %+v", cmd)
	}
}
