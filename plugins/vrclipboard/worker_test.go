package vrclipboard

import (
	"bytes"
	"context"
	"errors"
	"image"
	"io"
	"strings"
	"testing"
	"time"

	"sound-snoofer/internal/desktopinput"
	"sound-snoofer/internal/vroverlay"
)

type fakePad struct {
	active  bool
	batches [][]vroverlay.Event
	err     error
	draws   int
	polled  int
}

func (f *fakePad) Poll() ([]vroverlay.Event, error) {
	f.polled++
	if f.err != nil {
		return nil, f.err
	}
	if len(f.batches) == 0 {
		return nil, nil
	}
	events := f.batches[0]
	f.batches = f.batches[1:]
	return events, nil
}
func (f *fakePad) Active() bool           { return f.active }
func (f *fakePad) Draw(*image.RGBA) error { f.draws++; return nil }
func (f *fakePad) Close() error           { return nil }

func TestWorkerDispatchesOneFreshGestureWithoutReplayingFailure(t *testing.T) {
	for _, failure := range []error{nil, errors.New("input not confirmed")} {
		r, err := newPadRenderer()
		if err != nil {
			t.Fatal(err)
		}
		f := &fakePad{active: true, batches: [][]vroverlay.Event{
			{pointer(vroverlay.Move, 2), pointer(vroverlay.Down, 2)}, {pointer(vroverlay.Up, 2)}, nil, nil, {pointer(vroverlay.Up, 2)},
		}}
		ticks := make(chan time.Time, 5)
		for n := range 5 {
			ticks <- time.Unix(100, int64(n)*int64(50*time.Millisecond))
		}
		close(ticks)
		calls := 0
		desktop := desktopIO{capture: func() (desktopinput.Target, error) { return testTarget(), nil },
			send: func(_ context.Context, target desktopinput.Target, action desktopinput.Action) error {
				calls++
				if target != testTarget() || action != desktopinput.Paste {
					t.Fatal(target, action)
				}
				return failure
			}}
		var status bytes.Buffer
		err = runPanel(context.Background(), f, r, desktop, desktopinput.Target{}, ticks, &status)
		r.close()
		if err != nil || calls != 1 || !strings.Contains(status.String(), "ready") {
			t.Fatal(err, calls, status.String())
		}
		if f.draws > 4 {
			t.Fatal("redrew unchanged textures", f.draws)
		}
	}
}

func TestWorkerCancellationQuitAndPollFailureNeverSend(t *testing.T) {
	for _, mode := range []string{"cancel", "quit", "shown-quit", "poll"} {
		t.Run(mode, func(t *testing.T) {
			r, err := newPadRenderer()
			if err != nil {
				t.Fatal(err)
			}
			defer r.close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f := &fakePad{active: true, batches: [][]vroverlay.Event{{{Kind: vroverlay.Quit}, pointer(vroverlay.Move, 2), pointer(vroverlay.Down, 2), pointer(vroverlay.Up, 2)}}}
			if mode == "shown-quit" {
				f.batches[0] = []vroverlay.Event{{Kind: vroverlay.Shown}, pointer(vroverlay.Move, 2), pointer(vroverlay.Down, 2), pointer(vroverlay.Up, 2), {Kind: vroverlay.Quit}}
			}
			if mode == "cancel" {
				cancel()
			}
			if mode == "poll" {
				f.err = errors.New("overflow")
			}
			ticks := make(chan time.Time, 1)
			ticks <- time.Unix(100, 0)
			close(ticks)
			calls := 0
			desktop := desktopIO{capture: func() (desktopinput.Target, error) { return testTarget(), nil },
				send: func(context.Context, desktopinput.Target, desktopinput.Action) error { calls++; return nil }}
			var status bytes.Buffer
			err = runPanel(ctx, f, r, desktop, testTarget(), ticks, &status)
			if (mode == "poll") != (err != nil) || calls != 0 {
				t.Fatal(err, calls)
			}
			if (mode == "quit" || mode == "shown-quit") && !strings.Contains(status.String(), "SteamVR closed the overlay") {
				t.Fatal("lost Quit status", status.String())
			}
			if mode == "cancel" && (f.draws != 0 || f.polled != 0) {
				t.Fatal("cancelled worker touched backend")
			}
		})
	}
}

func TestWorkerPipeClosureCancelsAndReaderJoins(t *testing.T) {
	reader, writer := io.Pipe()
	ctx, stop := shutdownContext(context.Background(), reader)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("EOF did not cancel")
	}
	stop()
	reader, writer = io.Pipe()
	ctx, stop = shutdownContext(context.Background(), reader)
	stop()
	if ctx.Err() == nil {
		t.Fatal("stop did not cancel")
	}
	_ = writer.Close()
}

func TestWorkerExpiresQueuedGestureAfterSlowDispatch(t *testing.T) {
	r, err := newPadRenderer()
	if err != nil {
		t.Fatal(err)
	}
	defer r.close()
	f := &fakePad{active: true, batches: [][]vroverlay.Event{{
		pointer(vroverlay.Move, 2),
		pointer(vroverlay.Down, 2), pointer(vroverlay.Up, 2),
		pointer(vroverlay.Down, 2), pointer(vroverlay.Up, 2),
	}}}
	ticks := make(chan time.Time, 1)
	ticks <- time.Now()
	close(ticks)
	calls := 0
	desktop := desktopIO{
		capture: func() (desktopinput.Target, error) { return testTarget(), nil },
		send: func(context.Context, desktopinput.Target, desktopinput.Action) error {
			calls++
			if calls == 1 {
				time.Sleep(maxEventAge + 20*time.Millisecond)
			}
			return nil
		},
	}
	if err := runPanel(context.Background(), f, r, desktop, testTarget(), ticks, io.Discard); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expired queued gesture sent: %d calls", calls)
	}
}

func TestWorkerFollowsActiveTargetAndDiscardsQueuedGestures(t *testing.T) {
	r, err := newPadRenderer()
	if err != nil {
		t.Fatal(err)
	}
	defer r.close()
	f := &fakePad{active: true, batches: [][]vroverlay.Event{
		{pointer(vroverlay.Move, 2), pointer(vroverlay.Down, 2)}, // Held on the original target.
		{pointer(vroverlay.Up, 2), pointer(vroverlay.Move, 2), pointer(vroverlay.Down, 2), pointer(vroverlay.Up, 2)},
		{pointer(vroverlay.Move, 2), pointer(vroverlay.Down, 2)}, // Fresh gesture after the new target is drawn.
		{pointer(vroverlay.Up, 2)},
		{pointer(vroverlay.Move, 2), pointer(vroverlay.Down, 2), pointer(vroverlay.Up, 2)},
	}}
	ticks := make(chan time.Time, 5)
	for range 5 {
		ticks <- time.Now()
	}
	close(ticks)
	next := testTarget()
	next.Window++
	next.Label = "other.exe"
	captures, calls, firstSendDraws := 0, 0, 0
	desktop := desktopIO{
		capture: func() (desktopinput.Target, error) {
			captures++
			if captures == 1 {
				return testTarget(), nil
			}
			if captures == 4 {
				return desktopinput.Target{}, errors.New("excluded foreground")
			}
			return next, nil
		},
		send: func(_ context.Context, target desktopinput.Target, _ desktopinput.Action) error {
			calls++
			if f.polled != calls+3 || target != next {
				t.Fatalf("old or queued target gesture dispatched: poll=%d target=%+v", f.polled, target)
			}
			if calls == 1 {
				firstSendDraws = f.draws
			}
			return nil
		},
	}
	if err := runPanel(context.Background(), f, r, desktop, testTarget(), ticks, io.Discard); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("fresh target gesture calls=%d", calls)
	}
	if f.draws != firstSendDraws+1 {
		t.Fatal("unchanged target reset feedback or redrew the same frame", firstSendDraws, f.draws)
	}
}

func TestWorkerRetainsQuitWhileDiscardingTargetChangeBatch(t *testing.T) {
	r, err := newPadRenderer()
	if err != nil {
		t.Fatal(err)
	}
	defer r.close()
	f := &fakePad{active: true, batches: [][]vroverlay.Event{
		nil,
		{pointer(vroverlay.Move, 2), pointer(vroverlay.Down, 2), pointer(vroverlay.Up, 2), {Kind: vroverlay.Quit}},
	}}
	ticks := make(chan time.Time, 3)
	for range 3 {
		ticks <- time.Now()
	}
	close(ticks)
	captures := 0
	desktop := desktopIO{
		capture: func() (desktopinput.Target, error) {
			captures++
			target := testTarget()
			if captures > 1 {
				target.Window++
			}
			return target, nil
		},
		send: func(context.Context, desktopinput.Target, desktopinput.Action) error {
			t.Fatal("Quit batch dispatched input")
			return nil
		},
	}
	var status bytes.Buffer
	if err := runPanel(context.Background(), f, r, desktop, testTarget(), ticks, &status); err != nil {
		t.Fatal(err)
	}
	if captures != 2 || !strings.Contains(status.String(), "SteamVR closed the overlay") {
		t.Fatal("target change discarded Quit", captures, status.String())
	}
}
