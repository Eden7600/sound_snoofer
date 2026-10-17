package vroverlay

import (
	"errors"
	"image"
	"math"
	"reflect"
	"testing"
	"time"
	"unsafe"
)

// Every test uses this fake; no test calls Open or the installed runtime.
type fakeDriver struct {
	calls                []string
	fail                 string
	events               map[uint64][]nativeEvent
	draws                []uint64
	pixels               []byte
	activeValue          bool
	destroyErr, closeErr error
}

func (d *fakeDriver) step(name string) error {
	d.calls = append(d.calls, name)
	if d.fail == name {
		return errors.New(name + " failed")
	}
	return nil
}
func (d *fakeDriver) create(string) (uint64, uint64, error) { return 11, 12, d.step("create") }
func (d *fakeDriver) configure(uint64, int, int) error      { return d.step("configure") }
func (d *fakeDriver) draw(h uint64, p []byte, _, _ int) error {
	d.draws = append(d.draws, h)
	d.pixels = append([]byte(nil), p...)
	return d.step("draw")
}
func (d *fakeDriver) poll(h uint64, event *nativeEvent) bool {
	if len(d.events[h]) == 0 {
		return false
	}
	*event = d.events[h][0]
	d.events[h] = d.events[h][1:]
	return true
}
func (d *fakeDriver) active(uint64) bool { return d.activeValue }
func (d *fakeDriver) destroy(h uint64) error {
	if h != 11 {
		panic("destroying thumbnail or unrelated overlay")
	}
	d.step("destroy")
	return d.destroyErr
}
func (d *fakeDriver) close() error { d.step("close"); return d.closeErr }

func thumbnail() *image.RGBA { return image.NewRGBA(image.Rect(0, 0, 128, 128)) }

func TestSessionOwnershipAndFailureCleanup(t *testing.T) {
	for _, fail := range []string{"", "create", "configure", "draw"} {
		t.Run(fail, func(t *testing.T) {
			d := &fakeDriver{fail: fail}
			s, err := newSession(d, "test", 1024, 512, thumbnail())
			if (err != nil) != (fail != "") {
				t.Fatalf("session=%v error=%v", s, err)
			}
			if fail == "" {
				if !reflect.DeepEqual(d.draws, []uint64{12}) {
					t.Fatal("thumbnail not initialized")
				}
				d.activeValue = true
				if !s.Active() {
					t.Fatal("not active")
				}
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
				before := len(d.calls)
				if err := s.Close(); err != nil || len(d.calls) != before {
					t.Fatal("close repeated")
				}
				if s.Active() {
					t.Fatal("closed session active")
				}
				if _, err := s.Poll(); !errors.Is(err, ErrClosed) {
					t.Fatal(err)
				}
				if err := s.Draw(nil); !errors.Is(err, ErrClosed) {
					t.Fatal(err)
				}
			}
			if got := d.calls[len(d.calls)-2:]; !reflect.DeepEqual(got, []string{"destroy", "close"}) {
				t.Fatal(d.calls)
			}
		})
	}
}

func TestCleanupPreservesBothErrors(t *testing.T) {
	destroyErr, closeErr := errors.New("destroy"), errors.New("close")
	d := &fakeDriver{destroyErr: destroyErr, closeErr: closeErr}
	s, err := newSession(d, "test", 1024, 512, thumbnail())
	if err != nil {
		t.Fatal(err)
	}
	err = s.Close()
	if !errors.Is(err, destroyErr) || !errors.Is(err, closeErr) {
		t.Fatal(err)
	}
}

func mouse(kind uint32, x, y float32) nativeEvent {
	return nativeEvent{Type: kind, Age: .125, Data: [6]uint64{uint64(math.Float32bits(x)) | uint64(math.Float32bits(y))<<32, 1 | uint64(7)<<32}}
}

func TestEventsDecodeAndThumbnailIsolation(t *testing.T) {
	d := &fakeDriver{events: map[uint64][]nativeEvent{
		11: {mouse(301, 30, 40), {Type: 304}, {Type: 999}},
		12: {mouse(301, 30, 40), {Type: 700}},
	}}
	s := &Session{driver: d, main: 11, thumbnail: 12, height: 512}
	events, err := s.Poll()
	if err != nil || len(events) != 3 {
		t.Fatalf("%+v %v", events, err)
	}
	want := Event{Kind: Down, X: 30, Y: 472, Button: 1, Cursor: 7, Age: 125 * time.Millisecond}
	if events[0] != want || events[1].Kind != FocusLost || events[2].Kind != Quit {
		t.Fatal(events)
	}
	if events, err := s.Poll(); err != nil || len(events) != 0 {
		t.Fatal("events replayed")
	}
}

func TestPrimaryControllerChangePreservesCancellationAndDeviceIdentity(t *testing.T) {
	down, up := mouse(301, 30, 40), mouse(302, 30, 40)
	down.Device, up.Device = 3, 4
	// Both physical hands can occupy primary cursor slot zero.
	down.Data[1], up.Data[1] = 1, 1
	d := &fakeDriver{events: map[uint64][]nativeEvent{
		11: {down, {Type: 525}, up},
	}}
	s := &Session{driver: d, main: 11, thumbnail: 12, height: 512}
	events, err := s.Poll()
	if err != nil || len(events) != 3 {
		t.Fatalf("%+v %v", events, err)
	}
	if events[0].Device != 3 || events[2].Device != 4 || events[0].Cursor != 0 || events[2].Cursor != 0 || events[1].Kind != FocusLost {
		t.Fatalf("controller change lost: %+v", events)
	}
}
func TestInvalidNativeAgesAreAlwaysStale(t *testing.T) {
	for _, age := range []float32{-1, float32(math.NaN()), float32(math.Inf(1)), 4000} {
		if got := (nativeEvent{Age: age}).event(512).Age; got <= 250*time.Millisecond {
			t.Fatalf("age %v became %v", age, got)
		}
	}
}

func TestBoundedPollDiscardsEntireFailedBatch(t *testing.T) {
	for _, imageFailure := range []bool{false, true} {
		queue := make([]nativeEvent, maxEvents+10)
		for n := range queue {
			queue[n] = mouse(301, 1, 1)
		}
		if imageFailure {
			queue[1].Type = 517
		}
		d := &fakeDriver{events: map[uint64][]nativeEvent{11: queue}}
		s := &Session{driver: d, main: 11, thumbnail: 12, height: 512}
		events, err := s.Poll()
		if err == nil || len(events) != 0 {
			t.Fatalf("%+v %v", events, err)
		}
		if !imageFailure && (!errors.Is(err, ErrEventFlood) || len(d.events[11]) != 10) {
			t.Fatal("poll budget exceeded", err)
		}
	}
}

func TestDrawCopiesSubimageAndRejectsMalformedBuffers(t *testing.T) {
	d := &fakeDriver{}
	s := &Session{driver: d, main: 11, width: 2, height: 2}
	base := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for n := range base.Pix {
		base.Pix[n] = byte(n)
	}
	img := base.SubImage(image.Rect(1, 1, 3, 3)).(*image.RGBA)
	if err := s.Draw(img); err != nil {
		t.Fatal(err)
	}
	want := append(append([]byte(nil), base.Pix[20:28]...), base.Pix[36:44]...)
	if !reflect.DeepEqual(d.pixels, want) || !reflect.DeepEqual(d.draws, []uint64{11}) {
		t.Fatal(d.pixels, d.draws)
	}
	for _, bad := range []*image.RGBA{nil, image.NewRGBA(image.Rect(0, 0, 3, 2)), {Rect: image.Rect(0, 0, 2, 2)}, {Rect: image.Rect(0, 0, 2, 2), Stride: math.MaxInt, Pix: make([]byte, 8)}} {
		if err := s.Draw(bad); err == nil {
			t.Fatal("accepted invalid image")
		}
	}
	if len(d.draws) != 1 {
		t.Fatal("invalid image reached native call")
	}
}

func TestImageDimensionsAndEventABI(t *testing.T) {
	if err := validateImages(1024, 512, thumbnail()); err != nil {
		t.Fatal(err)
	}
	for _, size := range [][2]int{{0, 512}, {1025, 512}, {1024, 0}, {1024, 1025}} {
		if err := validateImages(size[0], size[1], thumbnail()); err == nil {
			t.Fatal(size)
		}
	}
	if validateImages(1024, 512, nil) == nil {
		t.Fatal("nil thumbnail accepted")
	}
	var e nativeEvent
	if unsafe.Sizeof(e) != 64 || unsafe.Offsetof(e.Data) != 16 || unsafe.Offsetof(e.Age) != 8 {
		t.Fatal("VREvent_t ABI drift")
	}
	if (nativeEvent{}).event(512).Kind != Unknown {
		t.Fatal("zero event becomes input")
	}
}
