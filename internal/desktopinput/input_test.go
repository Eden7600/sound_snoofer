package desktopinput

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

// These tests call only the injected implementation, never Capture, Send or native methods.
type fakeSystem struct {
	info                      windowInfo
	self, level               uint32
	foregroundWindow          uintptr
	environmentErr, windowErr error
	held                      map[uint16]bool
	activateOK                bool
	activations, pauses       int
	batches                   [][]keyEvent
	counts                    []int
	injectionErr              error
	before                    func(string)
}

func fake() *fakeSystem {
	return &fakeSystem{info: windowInfo{Target: Target{Window: 10, Process: 20, Thread: 30, Started: 40, Label: "editor.exe"}, Integrity: 0x2000, Visible: true}, self: 1, level: 0x2000, foregroundWindow: 10, held: map[uint16]bool{}, activateOK: true}
}
func (f *fakeSystem) step(name string) {
	if f.before != nil {
		f.before(name)
	}
}
func (f *fakeSystem) environment() (uint32, uint32, error) {
	f.step("environment")
	return f.self, f.level, f.environmentErr
}
func (f *fakeSystem) foreground() uintptr {
	f.step("foreground")
	return f.foregroundWindow
}
func (f *fakeSystem) window(uintptr) (windowInfo, error) {
	f.step("window")
	return f.info, f.windowErr
}
func (f *fakeSystem) activate(uintptr) bool {
	f.activations++
	f.step("activate")
	return f.activateOK
}
func (f *fakeSystem) down(key uint16) bool {
	f.step("down")
	return f.held[key]
}
func (f *fakeSystem) inject(keys []keyEvent) (int, error) {
	f.step("inject")
	f.batches = append(f.batches, append([]keyEvent(nil), keys...))
	if len(f.counts) != 0 {
		n := f.counts[0]
		f.counts = f.counts[1:]
		return n, f.injectionErr
	}
	return len(keys), nil
}
func (f *fakeSystem) pause(ctx context.Context, _ time.Duration) error {
	f.pauses++
	f.step("pause")
	return ctx.Err()
}

func TestShortcutsSendOneBalancedBatch(t *testing.T) {
	for _, tc := range []struct {
		action Action
		want   []keyEvent
	}{
		{Copy, []keyEvent{{0xa2, false}, {'C', false}, {'C', true}, {0xa2, true}}},
		{Cut, []keyEvent{{0xa2, false}, {'X', false}, {'X', true}, {0xa2, true}}},
		{Paste, []keyEvent{{0xa2, false}, {'V', false}, {'V', true}, {0xa2, true}}},
		{SelectAll, []keyEvent{{0xa2, false}, {'A', false}, {'A', true}, {0xa2, true}}},
		{Enter, []keyEvent{{0x0d, false}, {0x0d, true}}},
		{Tab, []keyEvent{{0x09, false}, {0x09, true}}},
		{Backspace, []keyEvent{{0x08, false}, {0x08, true}}},
	} {
		t.Run(string(tc.action), func(t *testing.T) {
			api := fake()
			if err := send(context.Background(), api, api.info.Target, tc.action); err != nil {
				t.Fatal(err)
			}
			if len(api.batches) != 1 || !reflect.DeepEqual(api.batches[0], tc.want) || api.activations != 0 {
				t.Fatal(api.batches, api.activations)
			}
		})
	}
}

func TestCaptureRejectsIneligibleOrChangingForeground(t *testing.T) {
	for _, mode := range []string{"valid", "none", "own", "host", "vrmonitor", "vrdashboard", "vrcompositor", "vrserver", "vrwebhelper", "hidden", "pid", "thread", "birth", "label", "higher-integrity", "token-error", "secure-desktop", "changed"} {
		t.Run(mode, func(t *testing.T) {
			api := fake()
			switch mode {
			case "none":
				api.foregroundWindow = 0
			case "own":
				api.info.Process = api.self
			case "host":
				api.info.Label = "SNOOFER.EXE"
			case "vrmonitor", "vrdashboard", "vrcompositor", "vrserver", "vrwebhelper":
				api.info.Label = mode + ".exe"
			case "hidden":
				api.info.Visible = false
			case "pid":
				api.info.Process = 0
			case "thread":
				api.info.Thread = 0
			case "birth":
				api.info.Started = 0
			case "label":
				api.info.Label = ""
			case "higher-integrity":
				api.info.Integrity = 0x3000
			case "token-error":
				api.windowErr = errors.New("cannot read token")
			case "secure-desktop":
				api.environmentErr = errors.New("desktop locked")
			case "changed":
				reads := 0
				api.before = func(name string) {
					if name == "foreground" {
						reads++
						if reads == 2 {
							api.foregroundWindow++
						}
					}
				}
			}
			target, err := capture(api)
			if (err == nil) != (mode == "valid") {
				t.Fatal(target, err)
			}
			if mode == "valid" && target != api.info.Target {
				t.Fatal(target)
			}
			if len(api.batches) != 0 || api.activations != 0 {
				t.Fatal("capture performed input")
			}
		})
	}
}

func TestSendRechecksEveryIdentityFieldBeforeInput(t *testing.T) {
	for _, late := range []bool{false, true} {
		for _, field := range []string{"window", "process", "thread", "started", "label", "visible", "privilege", "desktop"} {
			t.Run(field+map[bool]string{false: "/early", true: "/late"}[late], func(t *testing.T) {
				api := fake()
				target := api.info.Target
				change := func() {
					switch field {
					case "window":
						api.info.Window++
					case "process":
						api.info.Process++
					case "thread":
						api.info.Thread++
					case "started":
						api.info.Started++
					case "label":
						api.info.Label = "replacement.exe"
					case "visible":
						api.info.Visible = false
					case "privilege":
						api.info.Integrity++
					case "desktop":
						api.environmentErr = errors.New("desktop changed")
					}
				}
				if late {
					checks := 0
					api.before = func(name string) {
						if name == "environment" {
							checks++
							if checks == 2 {
								change()
							}
						}
					}
				} else {
					change()
				}
				if err := send(context.Background(), api, target, Paste); err == nil {
					t.Fatal("changed target accepted")
				}
				if len(api.batches) != 0 {
					t.Fatal("changed target received input")
				}
			})
		}
	}
}

func TestHeldKeysNeverGetReleasedOrOverwritten(t *testing.T) {
	for _, key := range []uint16{0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0x5b, 0x5c, 'V'} {
		for _, late := range []bool{false, true} {
			api := fake()
			if late {
				reads := 0
				api.before = func(name string) {
					if name == "window" {
						reads++
						if reads == 2 {
							api.held[key] = true
						}
					}
				}
			} else {
				api.held[key] = true
			}
			if err := send(context.Background(), api, api.info.Target, Paste); err == nil {
				t.Fatal("held key accepted", key, late)
			}
			if len(api.batches) != 0 || api.activations != 0 {
				t.Fatal("held key generated input", key)
			}
		}
	}
}

func TestForegroundRestorationIsOneBoundedAttempt(t *testing.T) {
	for _, mode := range []string{"immediate", "delayed", "denied", "never", "different", "changed-after-check"} {
		t.Run(mode, func(t *testing.T) {
			api := fake()
			api.foregroundWindow = 99
			if mode == "denied" {
				api.activateOK = false
			}
			api.before = func(name string) {
				if name == "activate" && mode == "immediate" {
					api.foregroundWindow = api.info.Window
				}
				if name == "pause" && (mode == "delayed" || mode == "changed-after-check") {
					api.foregroundWindow = api.info.Window
				}
				if name == "activate" && mode == "different" {
					api.foregroundWindow = 100
				}
				if name == "window" && mode == "changed-after-check" && api.activations != 0 {
					api.foregroundWindow = 100
				}
			}
			err := send(context.Background(), api, api.info.Target, Copy)
			want := mode == "immediate" || mode == "delayed"
			if (err == nil) != want || api.activations != 1 || api.pauses > 5 || (len(api.batches) == 1) != want {
				t.Fatal(mode, err, api.activations, api.pauses, api.batches)
			}
		})
	}
}

func TestCancellationAndUnknownActionNeverSend(t *testing.T) {
	for _, when := range []string{"before", "activate", "pause", "second-window", "final-foreground"} {
		t.Run(when, func(t *testing.T) {
			api := fake()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if when == "before" {
				cancel()
			}
			if when == "activate" || when == "pause" {
				api.foregroundWindow = 99
			}
			windowsRead, foregroundRead := 0, 0
			api.before = func(name string) {
				if name == "window" {
					windowsRead++
					if when == "second-window" && windowsRead == 2 {
						cancel()
					}
				}
				if name == "foreground" {
					foregroundRead++
					if when == "final-foreground" && foregroundRead == 2 {
						cancel()
					}
				}
				if name == when {
					cancel()
				}
			}
			if err := send(ctx, api, api.info.Target, Cut); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if len(api.batches) != 0 {
				t.Fatal("canceled shortcut sent")
			}
		})
	}
	api := fake()
	api.before = func(string) { t.Fatal("unknown action reached OS") }
	if send(context.Background(), api, api.info.Target, Action("delete")) == nil {
		t.Fatal("unknown action accepted")
	}
}

func TestPartialInputCleansOnlyInsertedDownKeysWithoutReplay(t *testing.T) {
	for n, want := range [][]keyEvent{nil, {{0xa2, true}}, {{'C', true}, {0xa2, true}}, {{0xa2, true}}} {
		api := fake()
		api.counts = []int{n}
		api.injectionErr = errors.New("input blocked")
		if err := send(context.Background(), api, api.info.Target, Copy); err == nil || !strings.Contains(err.Error(), "not confirmed") {
			t.Fatal(err)
		}
		if n == 0 {
			if len(api.batches) != 1 {
				t.Fatal("zero insertion generated cleanup", api.batches)
			}
		} else if len(api.batches) != 2 || !reflect.DeepEqual(api.batches[1], want) {
			t.Fatal(n, api.batches)
		}
	}
	api := fake()
	api.counts = []int{1}
	if send(context.Background(), api, api.info.Target, Enter) == nil || !reflect.DeepEqual(api.batches[1], []keyEvent{{0x0d, true}}) {
		t.Fatal(api.batches)
	}
}

func TestCleanupRemainsBoundedEvenWhenCancelledOrBlocked(t *testing.T) {
	api := fake()
	api.counts = []int{2, 0}
	api.injectionErr = errors.New("blocked")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api.before = func(name string) {
		if name == "inject" {
			cancel()
		}
	}
	err := send(ctx, api, api.info.Target, Paste)
	if err == nil || !strings.Contains(err.Error(), "key release not confirmed") || len(api.batches) != 2 {
		t.Fatal(err, api.batches)
	}
	for _, key := range api.batches[1] {
		if !key.Up {
			t.Fatal("cleanup replayed a key down")
		}
	}
}
