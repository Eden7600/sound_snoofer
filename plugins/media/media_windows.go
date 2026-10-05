//go:build windows

// Package media owns Windows media key commands.
package media

import (
	"context"
	"encoding/json"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"

	"sound-snoofer/snoofer"
)

type instance struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func (i *instance) Stop(ctx context.Context) error {
	i.cancel()
	select {
	case <-i.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Plugin registers media commands without claiming observed player state.
func Plugin() snoofer.Plugin {
	return snoofer.Plugin{ID: "media", Validate: func(raw json.RawMessage) error { return snoofer.DecodeSettings(raw, &struct{}{}) }, Label: "Windows media", Start: func(ctx context.Context, s snoofer.Services, raw json.RawMessage, _ map[string]snoofer.Instance) (snoofer.Instance, error) {
		if err := snoofer.DecodeSettings(raw, &struct{}{}); err != nil {
			return nil, err
		}
		runCtx, cancel := context.WithCancel(ctx)
		i := &instance{cancel: cancel, done: make(chan struct{})}
		queue := make(chan snoofer.Request, 8)
		controls := []snoofer.Control{}
		for _, name := range []string{"prev", "play", "next", "stop"} {
			controls = append(controls, snoofer.Control{ID: "media." + name, Label: "Media " + name, ShortLabel: name, Group: "Media", SurfaceOnly: true, Kind: "command", Icon: "media-" + name, Available: s.Live, Operations: []string{"press"}})
		}
		publish := func(status string) {
			for n := range controls {
				controls[n].Status = status
			}
			_ = s.Controls.Publish("media", controls, func(ctx context.Context, r snoofer.Request) error {
				select {
				case queue <- r:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				default:
					return fmt.Errorf("media queue full")
				}
			})
		}
		publish("")
		go func() {
			defer close(i.done)
			defer s.Controls.Remove("media")
			for {
				select {
				case <-runCtx.Done():
					return
				case r := <-queue:
					if runCtx.Err() != nil {
						return
					}
					if err := send(r.ID); err != nil {
						publish(err.Error())
					} else {
						publish("")
					}
				}
			}
		}()
		return i, nil
	}}
}
func send(id string) error {
	keys := map[string]uint16{"media.next": 0xb0, "media.prev": 0xb1, "media.stop": 0xb2, "media.play": 0xb3}
	key, ok := keys[id]
	if !ok {
		return fmt.Errorf("unknown media command")
	}
	type input struct {
		Kind, Padding    uint32
		Key, Scan        uint16
		Flags, Time, Pad uint32
		Extra            uintptr
		Tail             uint64
	}
	data := [2]input{{Kind: 1, Key: key}, {Kind: 1, Key: key, Flags: 2}}
	n, _, err := windows.NewLazySystemDLL("user32.dll").NewProc("SendInput").Call(2, uintptr(unsafe.Pointer(&data[0])), unsafe.Sizeof(data[0]))
	if n != 2 {
		return fmt.Errorf("media dispatch incomplete: %v", err)
	}
	return nil
}
