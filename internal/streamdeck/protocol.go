package streamdeck

import (
	"encoding/binary"
	"fmt"
)

const Vendor = 0x0fd9

// Keys and Encoders are the most keys and dials of any supported model; a
// Frame holds this many and each model uses the first of them.
const Keys = 36
const Encoders = 6

// Event is input, a connection or a failure from a deck. Device identifies
// the deck: its serial, or "path:" and its HID path when it has none. An
// Error without a Device is a discovery failure or no deck at all.
type Event struct {
	Generation uint64
	Device     string
	Serial     string
	Model      string // Model ID.
	Product    uint16 // USB product ID.
	Connected  bool
	Binding    string
	Key        int
	Encoder    int
	Delta      int
	Press      bool
	Release    bool // A key came up; only keys report releases.
	Error      string
}

// Decoder turns a model's input reports into edge events. Neo touch
// buttons are keys after the display keys.
type Decoder struct {
	Model                    Model
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
		count := d.Model.Keys() + d.Model.TouchKeys
		if count == 0 || length < count {
			return nil, fmt.Errorf("invalid key count")
		}
		states := b[4 : 4+count] // Extra payload bytes are ignored.
		for n, v := range states {
			if v > 1 {
				return nil, fmt.Errorf("invalid key state")
			}
			if d.keysReady && v == 1 && !d.keys[n] {
				events = append(events, Event{Key: n, Encoder: -1, Press: true})
			}
			if d.keysReady && v == 0 && d.keys[n] {
				events = append(events, Event{Key: n, Encoder: -1, Release: true})
			}
		}
		for n, v := range states {
			d.keys[n] = v == 1
		}
		d.keysReady = true
	case 3:
		count := d.Model.Dials
		if count == 0 || length < count+1 {
			return nil, fmt.Errorf("invalid encoder count")
		}
		states := b[5 : 5+count]
		switch b[4] {
		case 0:
			for n, v := range states {
				if v > 1 {
					return nil, fmt.Errorf("invalid encoder state")
				}
				if d.encodersReady && v == 1 && !d.encoders[n] {
					events = append(events, Event{Key: -1, Encoder: n, Press: true})
				}
			}
			for n, v := range states {
				d.encoders[n] = v == 1
			}
			d.encodersReady = true
		case 1:
			for n, v := range states {
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

// ImageReports splits a JPEG into output reports: a key image (0x07) or,
// with touch, the + XL strip or Neo screen (0x0B).
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

// displayReports splits a strip or screen image for model m.
func displayReports(m Model, image []byte, size int) ([][]byte, error) {
	if m.display == plusStrip {
		return lcdReports(0, 0, m.Dials*panelWidth, 100, image, size)
	}
	return ImageReports(0, true, image, size)
}

// lcdHeader is the size of a Stream Deck + strip report header.
const lcdHeader = 16

// lcdReports splits a JPEG for a region of the Stream Deck + strip (0x0C):
// x, y, width, height, last, chunk, then the payload length.
func lcdReports(x, y, w, h int, jpeg []byte, size int) ([][]byte, error) {
	if size < 32 || size > 8192 || len(jpeg) > 2*1024*1024 || x < 0 || y < 0 || w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid strip transfer")
	}
	out := [][]byte{}
	chunk := 0
	for len(jpeg) > 0 {
		n := min(size-lcdHeader, len(jpeg))
		b := make([]byte, size)
		b[0] = 2
		b[1] = 0x0c
		binary.LittleEndian.PutUint16(b[2:4], uint16(x))
		binary.LittleEndian.PutUint16(b[4:6], uint16(y))
		binary.LittleEndian.PutUint16(b[6:8], uint16(w))
		binary.LittleEndian.PutUint16(b[8:10], uint16(h))
		if n == len(jpeg) {
			b[10] = 1
		}
		binary.LittleEndian.PutUint16(b[11:13], uint16(chunk))
		binary.LittleEndian.PutUint16(b[13:15], uint16(n))
		copy(b[lcdHeader:], jpeg[:n])
		jpeg = jpeg[n:]
		chunk++
		out = append(out, b)
	}
	return out, nil
}
