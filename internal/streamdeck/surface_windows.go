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

// Surface runs a session per connected supported deck until its context
// ends. Each deck shows the latest frame given for it; events are bounded.
type Surface struct {
	events chan Event
	done   chan struct{}
	mu     sync.Mutex
	decks  map[string]chan Frame // By device ID, while its session runs.
	latest map[string]Frame      // Last frame shown per device ID, kept across reconnects.
}

// hidDeck is an open deck as a session uses it; tests replace the HID device.
type hidDeck interface {
	io(ctx context.Context, b []byte, write bool) (int, error)
	setBrightness(percent int) error
	close()
	describe() deckInfo
}

// deckInfo is what a session needs to know about an open deck.
type deckInfo struct {
	input, output int // Report lengths.
	serial        string
	model         Model
	product       uint16
}

// StartSurface discovers decks every two seconds and owns their sessions
// until ctx ends; Done closes once every session has cleared its display and
// closed its device.
func StartSurface(ctx context.Context) *Surface {
	return startSurface(ctx, discover, func(path string) (hidDeck, error) { return openDevice(path) })
}

func startSurface(ctx context.Context, discoverPaths func() ([]string, error), open func(string) (hidDeck, error)) *Surface {
	s := &Surface{events: make(chan Event, 64), done: make(chan struct{}), decks: map[string]chan Frame{}, latest: map[string]Frame{}}
	go s.run(ctx, discoverPaths, open)
	return s
}

// Events delivers input, connections and failures; it closes before Done.
func (s *Surface) Events() <-chan Event { return s.events }

// Done closes when the surface has stopped.
func (s *Surface) Done() <-chan struct{} { return s.done }

// Show sets the frame a deck shows, replacing any frame it has not drawn yet.
func (s *Surface) Show(id string, f Frame) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest[id] = f
	frames, ok := s.decks[id]
	if !ok {
		return
	}
	select {
	case <-frames:
	default:
	}
	select {
	case frames <- f:
	default:
	}
}

// deckID identifies a deck by serial, or by HID path without one.
func deckID(serial, path string) string {
	if serial != "" {
		return serial
	}
	return "path:" + path
}

func (s *Surface) run(ctx context.Context, discoverPaths func() ([]string, error), open func(string) (hidDeck, error)) {
	var sessions sync.WaitGroup
	defer close(s.done)
	defer close(s.events)
	defer sessions.Wait()
	running := map[string]bool{}      // HID paths with a session.
	openErrors := map[string]string{} // Temporary failures, deduplicated by HID path.
	retire := func(path string) bool {
		if _, failed := openErrors[path]; !failed {
			return true
		}
		select {
		case s.events <- Event{Device: deckID("", path), Retired: true}:
			delete(openErrors, path)
			return true
		case <-ctx.Done():
			return false
		}
	}
	ended := make(chan string)
	for ctx.Err() == nil {
		paths, err := discoverPaths()
		if err == nil {
			// A failed enumeration is not evidence that a path disappeared.
			present := map[string]bool{}
			for _, path := range paths {
				present[path] = true
			}
			for path := range openErrors {
				if !present[path] && !retire(path) {
					return
				}
			}
		}
		for _, path := range paths {
			if running[path] {
				continue
			}
			d, openErr := open(path)
			if openErr != nil {
				message := openErr.Error()
				if previous, known := openErrors[path]; !known || previous != message {
					select {
					case s.events <- Event{Device: deckID("", path), Error: message}:
						openErrors[path] = message
					case <-ctx.Done():
						return
					}
				}
				continue
			}
			// Retire before connecting: a serial-less session reuses this path ID.
			if !retire(path) {
				d.close()
				return
			}
			info := d.describe()
			id := deckID(info.serial, path)
			frames := make(chan Frame, 1)
			s.mu.Lock()
			if _, taken := s.decks[id]; taken {
				// The same deck on another interface path; one session drives it.
				s.mu.Unlock()
				d.close()
				continue
			}
			s.decks[id] = frames
			initial := s.latest[id]
			s.mu.Unlock()
			running[path] = true
			sessions.Add(1)
			go func() {
				defer sessions.Done()
				surfaceSession(ctx, d, id, initial, frames, s.events)
				d.close()
				s.mu.Lock()
				delete(s.decks, id)
				s.mu.Unlock()
				select {
				case ended <- path:
				case <-ctx.Done():
				}
			}()
		}
		if len(running) == 0 {
			if err == nil {
				err = fmt.Errorf("Stream Deck disconnected")
			}
			select {
			case s.events <- Event{Error: err.Error()}:
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
			case path := <-ended:
				delete(running, path)
			case <-timer.C:
				break wait
			}
		}
	}
}

func surfaceSession(parent context.Context, d hidDeck, id string, latest Frame, frames <-chan Frame, events chan<- Event) {
	info := d.describe()
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
		decoder := Decoder{Model: info.model}
		for ctx.Err() == nil {
			b := make([]byte, info.input)
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
				event.Device, event.Serial, event.Model, event.Product = id, info.serial, info.model.ID, info.product
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
		if n < info.model.Keys() {
			return ImageReports(n, false, image, info.output)
		}
		return displayReports(info.model, image, info.output)
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
		tiles, display, moving := renderChangedFrame(info.model, shown, cache, time.Since(started))
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
	failed := func(err error) Event {
		return Event{Device: id, Serial: info.serial, Model: info.model.ID, Product: info.product, Error: err.Error()}
	}
	if err := draw(latest, false); err != nil {
		select {
		case events <- failed(err):
		case <-ctx.Done():
		}
		return
	}
	select {
	case events <- Event{Device: id, Serial: info.serial, Model: info.model.ID, Product: info.product, Connected: true}:
	case <-ctx.Done():
		return
	}
	for {
		select {
		case <-ctx.Done():
			// The parent has cancelled; use a separate short deadline to clear displays.
			clearCtx, stop := context.WithTimeout(context.Background(), time.Second)
			tiles, display := renderFrame(info.model, Frame{})
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
			case events <- failed(err):
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
			if err := draw(latest, true); err != nil {
				select {
				case events <- failed(err):
				case <-ctx.Done():
				}
				return
			}
		case frame := <-frames:
			latest = frame
			if err := draw(frame, false); err != nil {
				select {
				case events <- failed(err):
				case <-ctx.Done():
				}
				return
			}
			generation.Store(frame.Generation)
		}
	}
}
