// Package vroverlay owns one SteamVR dashboard overlay. It never reads the
// clipboard, injects input, or creates a desktop window.
package vroverlay

import (
	"errors"
	"image"
	"math"
	"time"
)

type EventKind uint8

const (
	Unknown EventKind = iota
	Move
	Down
	Up
	Shown
	Hidden
	FocusLost
	Quit
)

// Event coordinates use the top-left origin of the main image, in pixels.
// Device preserves the tracked-device identity; Cursor is only the laser slot.
// Invalid native ages are made stale. Callers must reject non-finite coordinates.
type Event struct {
	Kind           EventKind
	X, Y           float32
	Button, Cursor uint32
	Device         uint32
	Age            time.Duration
}

var (
	ErrClosed      = errors.New("VR clipboard overlay is closed")
	ErrUnsupported = errors.New("VR clipboard requires Windows amd64")
	ErrEventFlood  = errors.New("VR clipboard event queue overflowed; input discarded")
)

const (
	interfaceVersion = "FnTable:IVROverlay_028"
	maxEvents        = 64
)

// nativeEvent follows Valve's Windows VREvent_t layout, including union alignment.
type nativeEvent struct {
	Type, Device uint32
	Age          float32
	_            uint32
	Data         [6]uint64
}

func (e nativeEvent) event(height int) Event {
	kind := map[uint32]EventKind{300: Move, 301: Down, 302: Up, 304: FocusLost, 500: Shown, 501: Hidden, 525: FocusLost, 700: Quit}[e.Type]
	age := time.Hour
	if !math.IsNaN(float64(e.Age)) && !math.IsInf(float64(e.Age), 0) && e.Age >= 0 && e.Age < 3600 {
		age = time.Duration(float64(e.Age) * float64(time.Second))
	}
	return Event{Kind: kind, X: math.Float32frombits(uint32(e.Data[0])), Y: float32(height) - math.Float32frombits(uint32(e.Data[0]>>32)),
		Button: uint32(e.Data[1]), Cursor: uint32(e.Data[1] >> 32), Device: e.Device, Age: age}
}

type driver interface {
	create(key string) (main, thumbnail uint64, err error)
	configure(main uint64, width, height int) error
	draw(handle uint64, pixels []byte, width, height int) error
	poll(handle uint64, event *nativeEvent) bool
	active(handle uint64) bool
	destroy(handle uint64) error
	close() error
}

// Session is owned by the goroutine whose OS thread is locked before Open.
// All methods, including Close, must be called on that same thread. Native calls
// cannot be canceled; the owning subprocess provides the shutdown boundary.
type Session struct {
	driver          driver
	main, thumbnail uint64
	width, height   int
	mainPixels      []byte
	thumbnailPixels []byte
}

func validateImages(width, height int, thumbnail *image.RGBA) error {
	if width <= 0 || height <= 0 || width > 1024 || height > 1024 {
		return errors.New("VR clipboard image dimensions must be between 1 and 1024")
	}
	if thumbnail == nil || thumbnail.Bounds().Dx() != 128 || thumbnail.Bounds().Dy() != 128 {
		return errors.New("VR clipboard thumbnail must be 128 by 128")
	}
	return nil
}

func newSession(d driver, key string, width, height int, thumbnail *image.RGBA) (*Session, error) {
	s := &Session{driver: d, width: width, height: height}
	var err error
	s.main, s.thumbnail, err = d.create(key)
	if err == nil && (s.main == 0 || s.thumbnail == 0) {
		err = errors.New("OpenVR returned an invalid dashboard handle")
	}
	if err == nil {
		err = d.configure(s.main, width, height)
	}
	if err == nil {
		s.thumbnailPixels, err = rgbaPixels(thumbnail)
	}
	if err == nil {
		err = d.draw(s.thumbnail, s.thumbnailPixels, 128, 128)
	}
	if err != nil {
		return nil, errors.Join(err, s.Close())
	}
	return s, nil
}

// Poll drains at most 64 native events across both owned handles. An overflowing
// queue or image-load failure discards this entire batch; the caller must close
// the session on error. Thumbnail pointer events never become pad input.
func (s *Session) Poll() ([]Event, error) {
	if s.driver == nil {
		return nil, ErrClosed
	}
	var events []Event
	count := 0
	for _, handle := range []uint64{s.main, s.thumbnail} {
		for {
			var raw nativeEvent
			if !s.driver.poll(handle, &raw) {
				break
			}
			count++
			if count == maxEvents {
				return nil, ErrEventFlood
			}
			if raw.Type == 517 { // VREvent_ImageFailed.
				return nil, errors.New("OpenVR could not load the clipboard image")
			}
			e := raw.event(s.height)
			if e.Kind != Unknown && (handle == s.main || e.Kind == Quit) {
				events = append(events, e)
			}
		}
	}
	return events, nil
}

func (s *Session) Active() bool { return s.driver != nil && s.driver.active(s.main) }

// Draw submits a new complete image. Dimensions are fixed when the session opens.
func (s *Session) Draw(img *image.RGBA) error {
	if s.driver == nil {
		return ErrClosed
	}
	if img == nil || img.Bounds().Dx() != s.width || img.Bounds().Dy() != s.height {
		return errors.New("VR clipboard image dimensions changed")
	}
	pixels, err := rgbaPixels(img)
	if err != nil {
		return err
	}
	s.mainPixels = pixels
	return s.driver.draw(s.main, s.mainPixels, s.width, s.height)
}

func rgbaPixels(img *image.RGBA) ([]byte, error) {
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	if width <= 0 || height <= 0 || width > 1024 || height > 1024 || img.Stride < width*4 || len(img.Pix) < width*4 || (height > 1 && img.Stride > (len(img.Pix)-width*4)/(height-1)) {
		return nil, errors.New("invalid VR clipboard pixel buffer")
	}
	pixels := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		copy(pixels[y*width*4:(y+1)*width*4], img.Pix[y*img.Stride:y*img.Stride+width*4])
	}
	return pixels, nil
}

// Close releases the main overlay (which owns its thumbnail), then this OpenVR
// client and loader. It does not shut down SteamVR. Repeated calls do nothing.
func (s *Session) Close() error {
	if s.driver == nil {
		return nil
	}
	d := s.driver
	s.driver = nil
	var err error
	if s.main != 0 {
		err = d.destroy(s.main)
	}
	err = errors.Join(err, d.close())
	s.mainPixels, s.thumbnailPixels = nil, nil
	return err
}
