package vrclipboard

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"runtime"
	"time"

	"sound-snoofer/internal/desktopinput"
	"sound-snoofer/internal/vroverlay"
)

type padOverlay interface {
	Poll() ([]vroverlay.Event, error)
	Active() bool
	Draw(*image.RGBA) error
	Close() error
}

type desktopIO struct {
	capture func() (desktopinput.Target, error)
	send    func(context.Context, desktopinput.Target, desktopinput.Action) error
}

// RunWorker owns the OpenVR client only in the isolated child process.
func RunWorker(parent context.Context, input io.Reader, output io.Writer) (result error) {
	pipe, ok := input.(io.ReadCloser)
	if !ok {
		return fmt.Errorf("clipboard worker requires an owned shutdown pipe")
	}
	ctx, stop := shutdownContext(parent, pipe)
	defer stop()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if ctx.Err() != nil {
		return nil
	}
	renderer, err := newPadRenderer()
	if err != nil {
		return err
	}
	defer renderer.close()
	if err := writeWorkerStatus(output, workerStatus{State: "waiting", Detail: "Connecting to SteamVR"}); err != nil {
		return err
	}
	desktop := desktopIO{capture: desktopinput.Capture, send: desktopinput.Send}
	latest, _ := desktop.capture()
	session, err := vroverlay.Open(panelWidth, panelHeight, renderer.thumbnail())
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, session.Close()) }()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	return runPanel(ctx, session, renderer, desktop, latest, ticker.C, output)
}

// Closing the owned pipe joins the reader even when OpenVR exits before its parent.
func shutdownContext(parent context.Context, pipe io.ReadCloser) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		var data [1]byte
		_, _ = pipe.Read(data[:])
		cancel()
	}()
	return ctx, func() {
		cancel()
		_ = pipe.Close()
		<-done
	}
}

type frameState struct {
	target         string
	available      bool
	hover, pressed int
	message        string
	failed         bool
}

func (p *panel) frame() frameState {
	pressed := -1
	if p.press != nil {
		pressed = p.press.button
	}
	return frameState{p.target.Label, p.target.Window != 0, p.hover, pressed, p.message, p.failed}
}

func runPanel(ctx context.Context, overlay padOverlay, renderer *padRenderer, desktop desktopIO,
	latest desktopinput.Target, ticks <-chan time.Time, output io.Writer) error {
	p := newPanel()
	if ctx.Err() != nil {
		return nil
	}
	if err := overlay.Draw(renderer.render(p)); err != nil {
		return err
	}
	if err := writeWorkerStatus(output, workerStatus{State: "ready", Detail: "Open Clipboard in the SteamVR dashboard"}); err != nil {
		return err
	}
	lastFrame := p.frame()
	draw := func() error {
		if err := overlay.Draw(renderer.render(p)); err != nil {
			return err
		}
		lastFrame = p.frame()
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-ticks:
			if !ok || ctx.Err() != nil {
				return nil
			}
			if target, err := desktop.capture(); err == nil {
				latest = target
			}
			active := overlay.Active()
			if active && !p.active {
				p.activate(latest, time.Now())
				if err := draw(); err != nil {
					return err
				}
				p.since = time.Now()
			} else if !active {
				p.deactivate()
			}
			polledAt := time.Now()
			events, err := overlay.Poll()
			if err != nil {
				return err
			}
			// Terminal events survive discarding queued gestures after a target change.
			for _, event := range events {
				if event.Kind == vroverlay.Quit {
					p.deactivate()
					return writeWorkerStatus(output, workerStatus{State: "waiting", Detail: "SteamVR closed the overlay"})
				}
			}
			for _, event := range events {
				now := time.Now()
				event.Age += now.Sub(polledAt)
				if ctx.Err() != nil {
					return nil
				}
				if event.Kind == vroverlay.Shown {
					if overlay.Active() {
						p.activate(latest, now)
						if err := draw(); err != nil {
							return err
						}
						p.since = time.Now()
					}
					break
				}
				before := p.target
				request := p.handle(event, now, latest)
				if p.target != before {
					if err := draw(); err != nil {
						return err
					}
					p.since = time.Now()
					break // A new target needs a gesture made after its name was drawn.
				}
				if request == nil {
					continue
				}
				if !overlay.Active() {
					p.deactivate()
					continue
				}
				if event.Age+time.Since(now) > maxEventAge {
					continue
				}
				err := desktop.send(ctx, request.target, request.action)
				if ctx.Err() != nil {
					return nil
				}
				p.failed = err != nil
				if err != nil {
					p.message = err.Error()
				} else {
					p.message = "Shortcut sent"
				}
			}
			if p.active && p.frame() != lastFrame {
				if err := draw(); err != nil {
					return err
				}
			}
		}
	}
}
