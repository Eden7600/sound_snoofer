package streamdeck

import (
	"encoding/binary"
	"fmt"
)

const Vendor = 0x0fd9
const Product = 0x00c6
const Keys = 36
const Encoders = 6

type Event struct {
	Generation uint64
	Serial     string
	Connected  bool
	Binding    string
	Key        int
	Encoder    int
	Delta      int
	Press      bool
	Error      string
}
type Decoder struct {
	keys                     [Keys]bool
	encoders                 [Encoders]bool
	keysReady, encodersReady bool
}

func (d *Decoder) Decode(b []byte) ([]Event, error) {
	if len(b) < 4 || b[0] != 1 {
		return nil, fmt.Errorf("invalid HID input header")
	}
	length := int(binary.LittleEndian.Uint16(b[2:4]))
	if length > len(b)-4 {
		return nil, fmt.Errorf("truncated HID payload")
	}
	b = b[:4+length]
	events := []Event{}
	switch b[1] {
	case 0:
		if length != Keys {
			return nil, fmt.Errorf("invalid key count")
		}
		for n, v := range b[4:] {
			if v > 1 {
				return nil, fmt.Errorf("invalid key state")
			}
			if d.keysReady && v == 1 && !d.keys[n] {
				events = append(events, Event{Key: n, Encoder: -1, Press: true})
			}
		}
		for n, v := range b[4:] {
			d.keys[n] = v == 1
		}
		d.keysReady = true
	case 3:
		if length != Encoders+1 {
			return nil, fmt.Errorf("invalid encoder count")
		}
		switch b[4] {
		case 0:
			for n, v := range b[5:] {
				if v > 1 {
					return nil, fmt.Errorf("invalid encoder state")
				}
				if d.encodersReady && v == 1 && !d.encoders[n] {
					events = append(events, Event{Key: -1, Encoder: n, Press: true})
				}
			}
			for n, v := range b[5:] {
				d.encoders[n] = v == 1
			}
			d.encodersReady = true
		case 1:
			for n, v := range b[5:] {
				if v != 0 {
					events = append(events, Event{Key: -1, Encoder: n, Delta: int(int8(v))})
				}
			}
		default:
			return nil, fmt.Errorf("unknown encoder event")
		}
	case 2: // Touch gestures are intentionally unbound.
	default:
		return nil, fmt.Errorf("unknown HID command")
	}
	return events, nil
}
func ImageReports(index int, touch bool, jpeg []byte, size int) ([][]byte, error) {
	if size < 32 || size > 8192 || len(jpeg) > 2*1024*1024 || index < 0 || index >= Keys {
		return nil, fmt.Errorf("invalid image transfer")
	}
	out := [][]byte{}
	chunk := 0
	for len(jpeg) > 0 {
		n := min(size-8, len(jpeg))
		b := make([]byte, size)
		b[0] = 2
		b[1] = 7
		b[2] = byte(index)
		if touch {
			b[1] = 0x0b
			b[2] = 0
		}
		if n == len(jpeg) {
			b[3] = 1
		}
		binary.LittleEndian.PutUint16(b[4:6], uint16(n))
		binary.LittleEndian.PutUint16(b[6:8], uint16(chunk))
		copy(b[8:], jpeg[:n])
		jpeg = jpeg[n:]
		chunk++
		out = append(out, b)
	}
	return out, nil
}
