// Package insta360 controls an Insta360 Link 2 webcam's privacy, AI tracking
// and framing through its UVC extension units. It never opens a video stream.
package insta360

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"

	"sound-snoofer/internal/camera"
	"sound-snoofer/snoofer"
)

const (
	tick           = 500 * time.Millisecond
	readInterval   = 2 * time.Second
	presenceRetry  = 5 * time.Second
	verifyDeadline = 3 * time.Second
)

// device is one open camera, as the worker uses it.
type device interface {
	Get(set camera.GUID, selector uint32) ([]byte, error)
	Set(set camera.GUID, selector uint32, data []byte) error
	Close() error
}

// Plugin returns inert metadata; the camera is opened only by Start's worker.
func Plugin() snoofer.Plugin {
	return snoofer.Plugin{ID: "insta360", Label: "Insta360", Validate: validate, Start: func(ctx context.Context, s snoofer.Services, raw json.RawMessage, _ map[string]snoofer.Instance) (snoofer.Instance, error) {
		if err := validate(raw); err != nil {
			return nil, err
		}
		o := &nativeOpener{}
		return start(ctx, s, o.open, o.release), nil
	}}
}

func validate(raw json.RawMessage) error {
	return snoofer.DecodeSettings(raw, &struct{}{})
}

type instance struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error // Cleanup outcome, written before done closes.
}

// Stop cancels the worker and waits for it to close the camera.
func (i *instance) Stop(ctx context.Context) error {
	i.cancel()
	select {
	case <-i.done:
		return i.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func start(ctx context.Context, s snoofer.Services, open func() (device, error), release func() error) *instance {
	runCtx, cancel := context.WithCancel(ctx)
	i := &instance{cancel: cancel, done: make(chan struct{})}
	w := newWorker(s, open)
	go func() {
		defer close(i.done)
		defer s.Controls.Remove("insta360")
		// The camera's COM apartment belongs to this thread.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		w.run(runCtx)
		i.err = errors.Join(w.close(), release())
	}()
	return i
}

// pendingWrite is a requested value awaiting readback.
type pendingWrite struct {
	want     string
	deadline time.Time
}

type worker struct {
	services snoofer.Services
	open     func() (device, error)
	requests chan snoofer.Request

	dev          device
	lastOpen     time.Time
	lastRead     time.Time
	status       status
	framing      byte
	framingKnown bool
	pending      map[string]pendingWrite
	notes        map[string]string // Per-control outcome, such as Failed, until the next request.
	link         snoofer.ConnectionTracker
	absent       bool
}

func newWorker(s snoofer.Services, open func() (device, error)) *worker {
	return &worker{services: s, open: open, requests: make(chan snoofer.Request, 8), pending: map[string]pendingWrite{}, notes: map[string]string{}}
}

func (w *worker) run(ctx context.Context) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		w.step(time.Now())
		w.publish(time.Now())
		select {
		case <-ctx.Done():
			return
		case r := <-w.requests:
			w.handle(r, time.Now())
		case <-ticker.C:
		}
	}
}

// step keeps the camera open and its state fresh, and settles pending writes.
func (w *worker) step(now time.Time) {
	if w.dev == nil {
		if !w.lastOpen.IsZero() && now.Sub(w.lastOpen) < presenceRetry {
			return
		}
		w.lastOpen = now
		dev, err := w.open()
		if errors.Is(err, camera.ErrAbsent) {
			w.absent = true
			w.link.Observe(snoofer.ConnectionDisconnected, now)
			return
		}
		if err != nil {
			w.absent = false
			w.link.Observe(snoofer.ConnectionError, now)
			w.link.Fail(err.Error(), now)
			return
		}
		w.dev, w.absent = dev, false
		w.lastRead = time.Time{}
	}
	if len(w.pending) == 0 && !w.lastRead.IsZero() && now.Sub(w.lastRead) < readInterval {
		return
	}
	w.read(now)
	w.verify(now)
}

// read refreshes status and framing; any failure drops the device so the
// next step reopens it.
func (w *worker) read(now time.Time) {
	packet, err := w.dev.Get(xu1, selVideoMode)
	if err == nil {
		w.status = parseStatus(packet)
		var style []byte
		style, err = w.dev.Get(xu1, selComposition)
		if err == nil && len(style) > 0 {
			w.framing, w.framingKnown = style[0], true
		}
	}
	w.lastRead = now
	if err != nil {
		w.drop(err, now)
		return
	}
	w.link.Observe(snoofer.ConnectionConnected, now)
	w.link.Activity(now)
}

func (w *worker) drop(err error, now time.Time) {
	_ = w.dev.Close() // The device already failed; its close result adds nothing.
	w.dev = nil
	w.status, w.framingKnown = status{}, false
	w.pending = map[string]pendingWrite{}
	w.link.Observe(snoofer.ConnectionError, now)
	w.link.Fail(err.Error(), now)
}

// observed is a control's current value as read from the camera, or "".
func (w *worker) observed(id string) string {
	switch id {
	case "insta360.privacy":
		if w.status.PrivacyKnown {
			return onOff(w.status.Privacy)
		}
	case "insta360.tracking":
		if w.status.ModeKnown {
			return trackingName(w.status.Mode)
		}
	case "insta360.framing":
		if w.framingKnown {
			return framingName(w.framing)
		}
	}
	return ""
}

func (w *worker) verify(now time.Time) {
	for id, p := range w.pending {
		switch {
		case w.observed(id) == p.want:
			delete(w.pending, id)
		case now.After(p.deadline):
			delete(w.pending, id)
			w.notes[id] = "Failed"
		}
	}
}

// handle runs one request. Availability is enforced by Dispatch; the checks
// here cover state that changed since the request was made.
func (w *worker) handle(r snoofer.Request, now time.Time) {
	delete(w.notes, r.ID)
	if !w.services.Live {
		w.notes[r.ID] = "Preview"
		return
	}
	if w.dev == nil {
		return
	}
	if r.ID != "insta360.privacy" && w.status.Privacy {
		w.notes[r.ID] = "Privacy"
		return
	}
	// The camera refuses full-body framing while tracking a group.
	if r.ID == "insta360.framing" && r.Value == "full" && w.status.Mode == modeAutoFraming {
		w.notes[r.ID] = "Group"
		return
	}
	want, err := w.write(r)
	if err != nil {
		w.notes[r.ID] = "Failed"
		w.link.Fail(err.Error(), now)
		return
	}
	if want != "" {
		w.pending[r.ID] = pendingWrite{want: want, deadline: now.Add(verifyDeadline)}
	}
	w.read(now)
	if w.dev != nil {
		w.verify(now)
	}
}

// write sends a request to the camera and returns the value to verify, or ""
// for commands.
func (w *worker) write(r snoofer.Request) (string, error) {
	switch r.ID {
	case "insta360.privacy":
		want := !w.status.Privacy
		value := byte(0)
		if want {
			value = 1
		}
		return onOff(want), w.dev.Set(xu2, selPrivacy, []byte{value})
	case "insta360.tracking":
		mode, ok := trackingModes[r.Value]
		if !ok {
			return "", fmt.Errorf("unknown tracking %q", r.Value)
		}
		return r.Value, w.dev.Set(xu1, selVideoMode, videoModePacket(mode))
	case "insta360.framing":
		style, ok := framingStyles[r.Value]
		if !ok {
			return "", fmt.Errorf("unknown framing %q", r.Value)
		}
		if err := w.enableAIZoom(); err != nil {
			return "", err
		}
		return r.Value, w.dev.Set(xu1, selComposition, []byte{style})
	case "insta360.reset":
		if w.status.Mode == modeNormal {
			return "", w.dev.Set(xu1, selPanTilt, centrePanTilt())
		}
		return "", w.dev.Set(xu1, selBias, centreBias())
	}
	return "", fmt.Errorf("unknown control %q", r.ID)
}

// enableAIZoom sets the extended-function bit framing styles need.
func (w *worker) enableAIZoom() error {
	word, err := w.dev.Get(xu1, selExtended)
	if err != nil {
		return err
	}
	updated, changed, err := withAIZoom(word)
	if err != nil || !changed {
		return err
	}
	return w.dev.Set(xu1, selExtended, updated)
}

func (w *worker) close() error {
	if w.dev == nil {
		return nil
	}
	err := w.dev.Close()
	w.dev = nil
	return err
}

func onOff(on bool) string {
	if on {
		return "On"
	}
	return "Off"
}
