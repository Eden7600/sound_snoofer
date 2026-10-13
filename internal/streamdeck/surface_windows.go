//go:build windows

package streamdeck

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// StartSurface owns HID workers until done closes. Frames and events are bounded.
func StartSurface(ctx context.Context) (chan Frame, <-chan Event, <-chan struct{}) {
	return startSurface(ctx, discover)
}
func startSurface(ctx context.Context, discoverPaths func() ([]string, error)) (chan Frame, <-chan Event, <-chan struct{}) {
	frames := make(chan Frame, 1)
	events := make(chan Event, 64)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(events)
		var latest Frame
		for ctx.Err() == nil {
			paths, err := discoverPaths()
			var d *device
			if err == nil {
				for _, path := range paths {
					d, err = openDevice(path)
					if err == nil {
						break
					}
				}
			}
			if d != nil {
				surfaceSession(ctx, d, frames, events, &latest)
				d.close()
			} else {
				if err == nil {
					err = fmt.Errorf("Stream Deck disconnected")
				}
				select {
				case events <- Event{Error: err.Error()}:
				default:
				}
			}
			timer := time.NewTimer(2 * time.Second)
		wait:
			for {
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case latest = <-frames:
				case <-timer.C:
					break wait
				}
			}
		}
	}()
	return frames, events, done
}

func surfaceSession(parent context.Context, d *device, frames <-chan Frame, events chan<- Event, latest *Frame) {
	ctx, cancel := context.WithCancel(parent)
	var wg sync.WaitGroup
	defer wg.Wait()
	defer cancel()
	inputs := make(chan Event, 64)
	failures := make(chan error, 1)
	var generation atomic.Uint64
	generation.Store(latest.Generation)
	wg.Add(1)
	go func() {
		defer wg.Done()
		decoder := Decoder{Model: d.model}
		for ctx.Err() == nil {
			b := make([]byte, d.input)
			n, err := d.io(ctx, b, false)
			if err == context.DeadlineExceeded {
				continue
			}
			if err != nil {
				select {
				case failures <- err:
				default:
				}
				return
			}
			list, err := decoder.Decode(b[:n])
			if err != nil {
				continue
			}
			for _, event := range list {
				event.Generation = generation.Load()
				event.Serial = d.serial
				select {
				case inputs <- event:
				case <-ctx.Done():
					return
				default:
					select {
					case failures <- fmt.Errorf("input queue full"):
					default:
					}
					return
				}
			}
		}
	}()
	// reports splits image n of a frame: the keys, then the strip or screen.
	reports := func(n int, image []byte) ([][]byte, error) {
		if n < d.model.Keys() {
			return ImageReports(n, false, image, d.output)
		}
		return displayReports(d.model, image, d.output)
	}
	var previous [][]byte
	var lastFrame Frame
	cache := &keyCache{}
	started := time.Now() // Scroll phases count from the session's start.
	haveFrame := false
	brightness := -1 // Last backlight sent; -1 before any.
	// While any text scrolls, the scroll ticker redraws the shown frame; it
	// stops when nothing scrolls.
	scroll := time.NewTicker(scrollInterval)
	scroll.Stop()
	defer scroll.Stop()
	scrolling, ticking := false, false
	draw := func(frame Frame, tick bool) error {
		if haveFrame && frame == lastFrame && !tick {
			return nil
		}
		defer func() {
			if scrolling != ticking {
				ticking = scrolling
				if ticking {
					scroll.Reset(scrollInterval)
				} else {
					scroll.Stop()
				}
			}
		}()
		// A dark frame shows nothing: blank images, then the backlight off.
		// Otherwise the images come first, then the backlight.
		shown, want := frame, frame.Brightness
		if frame.Dark {
			shown, want = Frame{}, 0
		}
		defer func() {
			if want != brightness && (want > 0 || frame.Dark) {
				// The black frame already hides the deck; a device without the
				// brightness report only keeps its backlight.
				_ = d.setBrightness(want)
				brightness = want
			}
		}()
		tiles, display, moving := renderChangedFrame(d.model, shown, cache, time.Since(started))
		scrolling = moving
		all := append(tiles, display)
		for n, tile := range all {
			if tile == nil || len(previous) == len(all) && bytes.Equal(previous[n], tile) {
				continue
			}
			list, err := reports(n, tile)
			if err != nil {
				return err
			}
			for _, report := range list {
				if _, err = d.io(ctx, report, true); err != nil {
					return err
				}
			}
		}
		previous = all
		lastFrame = frame
		haveFrame = true
		return nil
	}
	if err := draw(*latest, false); err != nil {
		select {
		case events <- Event{Error: err.Error()}:
		case <-ctx.Done():
		}
		return
	}
	select {
	case events <- Event{Serial: d.serial, Model: d.model.ID, Connected: true}:
	case <-ctx.Done():
		return
	}
	for {
		select {
		case <-ctx.Done():
			// The parent has cancelled; use a separate short deadline to clear displays.
			clearCtx, stop := context.WithTimeout(context.Background(), time.Second)
			tiles, display := renderFrame(d.model, Frame{})
			for n, tile := range append(tiles, display) {
				if tile == nil {
					continue
				}
				list, err := reports(n, tile)
				if err != nil {
					break
				}
				for _, report := range list {
					if _, err = d.io(clearCtx, report, true); err != nil {
						break
					}
				}
				if clearCtx.Err() != nil {
					break
				}
			}
			stop()
			return
		case err := <-failures:
			select {
			case events <- Event{Error: err.Error()}:
			case <-ctx.Done():
			}
			return
		case event := <-inputs:
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		case <-scroll.C:
			if err := draw(*latest, true); err != nil {
				select {
				case events <- Event{Error: err.Error()}:
				case <-ctx.Done():
				}
				return
			}
		case frame := <-frames:
			*latest = frame
			if err := draw(frame, false); err != nil {
				select {
				case events <- Event{Error: err.Error()}:
				case <-ctx.Done():
				}
				return
			}
			generation.Store(frame.Generation)
		}
	}
}
