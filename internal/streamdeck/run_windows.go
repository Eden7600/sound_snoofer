//go:build windows

package streamdeck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/control"
	"sync"
	"time"
	"unsafe"
)

func Start(ctx context.Context, cfg *config.StreamDeck) (chan control.State, <-chan Event) {
	states := make(chan control.State, 1)
	events := make(chan Event, 64)
	go run(ctx, cfg, states, events)
	return states, events
}
func run(ctx context.Context, cfg *config.StreamDeck, states <-chan control.State, events chan Event) {
	defer close(events)
	var latest control.State
	for ctx.Err() == nil {
		paths, err := discover()
		var d *device
		if err == nil {
			for _, path := range paths {
				d, err = openDevice(path)
				if err == nil {
					break
				}
			}
		}
		if d == nil {
			if err != nil {
				select {
				case events <- Event{Error: err.Error()}:
				default:
				}
			}
			timer := time.NewTimer(2 * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case latest = <-states:
				timer.Stop()
			case <-timer.C:
			}
			continue
		}
		layout := append([]string(nil), bindings...)
		if cfg != nil && d.serial != "" {
			for _, profile := range cfg.Profiles {
				if profile.Serial == d.serial {
					layout = append([]string(nil), profile.Keys...)
				}
			}
		}
		serve(ctx, d, layout, states, events, &latest)
		d.close()
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}
func serve(ctx context.Context, d *device, layout []string, states <-chan control.State, events chan Event, initial *control.State) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	defer wg.Wait()
	// Cancellation must happen before joining workers.
	defer cancel()
	frames := make(chan control.State, 1)
	frames <- *initial
	fail := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		decoder := Decoder{}
		for ctx.Err() == nil {
			b := make([]byte, d.input)
			n, e := d.io(ctx, b, false)
			if errors.Is(e, context.DeadlineExceeded) {
				continue
			}
			if e != nil {
				fail <- e
				return
			}
			list, e := decoder.Decode(b[:n])
			if e != nil {
				continue
			}
			for _, event := range list {
				if event.Encoder < 0 {
					if event.Key >= len(layout) || layout[event.Key] == "" {
						continue
					}
					event.Binding = layout[event.Key]
				}
				select {
				case events <- event:
				case <-ctx.Done():
					return
				default:
					fail <- fmt.Errorf("Stream Deck input queue full; reconnecting without replay")
					return
				}
			}
		}
	}()
	go func() {
		defer wg.Done()
		last := ""
		var previous [][]byte
		for {
			select {
			case <-ctx.Done():
				return
			case state := <-frames:
				key, _ := json.Marshal(struct {
					Status          string
					Pending         bool
					Intent          any
					Numbers         any
					Recorder        any
					Live, Connected bool
				}{state.Notice, state.Plan != nil && state.Plan.HasChanges(), state.Intent, state.Snapshot.Numbers, state.Recorder, state.Live, state.Connected})
				if string(key) == last {
					continue
				}
				last = string(key)
				tiles, touch := display(state, layout)
				all := append(tiles, touch)
				for n, tile := range all {
					if len(previous) == len(all) && bytes.Equal(previous[n], tile) {
						continue
					}
					reports, e := ImageReports(n%Keys, n == Keys, tile, d.output)
					if e != nil {
						fail <- e
						return
					}
					for _, report := range reports {
						if _, e = d.io(ctx, report, true); e != nil {
							fail <- e
							return
						}
					}
				}
				previous = all
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-fail:
			select {
			case events <- Event{Error: err.Error()}:
			default:
			}
			return
		case state := <-states:
			*initial = state
			select {
			case <-frames:
			default:
			}
			select {
			case frames <- state:
			default:
			}
		}
	}
}

// Media sends a single balanced media keystroke; it does not infer player state.
func Media(command string) error {
	keys := map[string]uint16{"media-next": 0xb0, "media-prev": 0xb1, "media-stop": 0xb2, "media-play": 0xb3}
	key, ok := keys[command]
	if !ok {
		return fmt.Errorf("unknown media command")
	}
	// INPUT is 40 bytes on Windows amd64; union alignment begins at offset 8.
	type input struct {
		Kind    uint32
		Padding uint32
		Key     uint16
		Scan    uint16
		Flags   uint32
		Time    uint32
		Pad     uint32
		Extra   uintptr
		Tail    uint64
	}
	data := [2]input{{Kind: 1, Key: key}, {Kind: 1, Key: key, Flags: 2}}
	n, _, e := windows.NewLazySystemDLL("user32.dll").NewProc("SendInput").Call(2, uintptr(unsafe.Pointer(&data[0])), unsafe.Sizeof(data[0]))
	if n != 2 {
		return fmt.Errorf("media dispatch incomplete: %v", e)
	}
	return nil
}
