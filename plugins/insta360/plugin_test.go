package insta360

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"sound-snoofer/internal/camera"
	"sound-snoofer/snoofer"
)

// write is one Set call the fake camera received.
type write struct {
	set      camera.GUID
	selector uint32
	data     []byte
}

// fakeCamera models a Link 2: writes change its state unless frozen.
type fakeCamera struct {
	mode, framing byte
	privacy       bool
	extended      uint16
	frozen        bool // Accept writes without applying them.
	statusLength  int
	failGet       error
	writes        []write
	closed        bool
}

func (c *fakeCamera) Get(set camera.GUID, selector uint32) ([]byte, error) {
	if c.failGet != nil {
		return nil, c.failGet
	}
	switch {
	case set == xu1 && selector == selVideoMode:
		p := make([]byte, 0x38)
		p[0] = c.mode
		if c.mode != modeNormal {
			p[1] = 2 // Working.
		}
		if c.privacy {
			p[offsetFlags] = 1
		}
		return p[:c.statusLength], nil
	case set == xu1 && selector == selComposition:
		return []byte{c.framing}, nil
	case set == xu1 && selector == selExtended:
		return binary.LittleEndian.AppendUint16(nil, c.extended), nil
	}
	return nil, camera.ErrNoControl
}

func (c *fakeCamera) Set(set camera.GUID, selector uint32, data []byte) error {
	c.writes = append(c.writes, write{set, selector, append([]byte(nil), data...)})
	if c.frozen {
		return nil
	}
	switch {
	case set == xu2 && selector == selPrivacy:
		c.privacy = data[0] == 1
	case set == xu1 && selector == selVideoMode:
		c.mode = data[0]
	case set == xu1 && selector == selComposition:
		c.framing = data[0]
	case set == xu1 && selector == selExtended:
		c.extended = binary.LittleEndian.Uint16(data)
	}
	return nil
}

func (c *fakeCamera) Close() error {
	c.closed = true
	return nil
}

func newFake() *fakeCamera {
	return &fakeCamera{framing: 1, statusLength: 0x38}
}

func testWorker(live bool, cam *fakeCamera) (*worker, *time.Time) {
	s := snoofer.Services{Controls: snoofer.NewControls(), Live: live}
	w := newWorker(s, func() (device, error) {
		if cam == nil {
			return nil, camera.ErrAbsent
		}
		return cam, nil
	})
	now := time.Unix(1000, 0)
	w.step(now)
	return w, &now
}

func control(t *testing.T, w *worker, now time.Time, id string) snoofer.Control {
	t.Helper()
	for _, c := range w.controls(now) {
		if c.ID == id {
			return c
		}
	}
	t.Fatal("missing", id)
	return snoofer.Control{}
}

func TestParseStatusAcrossFirmwareLengths(t *testing.T) {
	// The 2026-10-07 live read: Normal, privacy flag clear.
	live := make([]byte, 0x38)
	binary.LittleEndian.PutUint16(live[offsetFlags:], 0x2A20)
	if s := parseStatus(live); !s.ModeKnown || s.Mode != modeNormal || !s.PrivacyKnown || s.Privacy {
		t.Fatal(s)
	}
	short := make([]byte, 0x2E)
	short[0] = modeAutoFraming
	if s := parseStatus(short); !s.ModeKnown || s.Mode != modeAutoFraming || s.PrivacyKnown {
		t.Fatal("short packet", s)
	}
	if s := parseStatus(nil); s.ModeKnown || s.PrivacyKnown {
		t.Fatal("empty packet", s)
	}
}

func TestVideoModePacketLeavesPoseUnchanged(t *testing.T) {
	p := videoModePacket(modeAutoComposition)
	if len(p) != 0x38 || p[0] != modeAutoComposition {
		t.Fatal(p)
	}
	for _, offset := range []int{0x26, 0x2A, 0x2E} {
		if binary.LittleEndian.Uint32(p[offset:]) != 3610 {
			t.Fatalf("pose at %#x: % x", offset, p[offset:offset+4])
		}
	}
	if int16(binary.LittleEndian.Uint16(p[0x34:])) != -450 || binary.LittleEndian.Uint16(p[0x32:]) != 0 {
		t.Fatal("zoom or pitch", p[0x32:0x36])
	}
}

func TestPrivacyVerifiedOrFailed(t *testing.T) {
	cam := newFake()
	w, now := testWorker(true, cam)
	if c := control(t, w, *now, "insta360.privacy"); !c.Available || c.Value != "Off" {
		t.Fatal(c)
	}
	w.handle(snoofer.Request{ID: "insta360.privacy", Operation: "press"}, *now)
	if !bytes.Equal(cam.writes[0].data, []byte{1}) || cam.writes[0].set != xu2 {
		t.Fatal(cam.writes)
	}
	if c := control(t, w, *now, "insta360.privacy"); c.Value != "On" || c.Status != "" {
		t.Fatal("verified privacy", c.Value, c.Status)
	}
	// Privacy gates the other controls.
	for _, id := range []string{"insta360.tracking", "insta360.framing", "insta360.reset"} {
		if c := control(t, w, *now, id); c.Available || c.Status != "Privacy" {
			t.Fatal(id, c.Available, c.Status)
		}
	}
	// A write the camera does not apply is Pending, then Failed.
	cam.frozen = true
	w.handle(snoofer.Request{ID: "insta360.privacy", Operation: "press"}, *now)
	if c := control(t, w, *now, "insta360.privacy"); c.Status != "Pending" {
		t.Fatal("pending", c.Status)
	}
	w.step(now.Add(4 * time.Second))
	if c := control(t, w, now.Add(4*time.Second), "insta360.privacy"); c.Status != "Failed" || c.Value != "On" {
		t.Fatal("failed", c.Value, c.Status)
	}
}

func TestTrackingAndFraming(t *testing.T) {
	cam := newFake()
	w, now := testWorker(true, cam)
	w.handle(snoofer.Request{ID: "insta360.tracking", Operation: "set", Value: "group"}, *now)
	if c := control(t, w, *now, "insta360.tracking"); c.Value != "group" || c.Status != "" {
		t.Fatal(c.Value, c.Status)
	}
	if c := control(t, w, *now, "insta360.state"); c.Value != "Working" {
		t.Fatal("state", c.Value)
	}
	// Full body is refused while tracking a group, without writing.
	writes := len(cam.writes)
	w.handle(snoofer.Request{ID: "insta360.framing", Operation: "set", Value: "full"}, *now)
	if len(cam.writes) != writes {
		t.Fatal("full body written in Group")
	}
	if c := control(t, w, *now, "insta360.framing"); c.Status != "Group" {
		t.Fatal(c.Status)
	}
	// Framing enables AI zoom first.
	w.handle(snoofer.Request{ID: "insta360.framing", Operation: "set", Value: "half"}, *now)
	last := cam.writes[len(cam.writes)-2:]
	if last[0].selector != selExtended || binary.LittleEndian.Uint16(last[0].data)&extendedAIZoom == 0 || last[1].selector != selComposition || last[1].data[0] != 2 {
		t.Fatal(last)
	}
	if c := control(t, w, *now, "insta360.framing"); c.Value != "half" || c.Status != "" {
		t.Fatal(c.Value, c.Status)
	}
	// AI zoom already on: no extra write.
	writes = len(cam.writes)
	w.handle(snoofer.Request{ID: "insta360.framing", Operation: "set", Value: "head"}, *now)
	if len(cam.writes) != writes+1 {
		t.Fatal("rewrote AI zoom", cam.writes[writes:])
	}
}

func TestResetByMode(t *testing.T) {
	cam := newFake()
	w, now := testWorker(true, cam)
	w.handle(snoofer.Request{ID: "insta360.reset", Operation: "press"}, *now)
	if last := cam.writes[len(cam.writes)-1]; last.selector != selPanTilt || !bytes.Equal(last.data, make([]byte, 8)) {
		t.Fatal("normal reset", last)
	}
	cam.mode = modeAutoComposition
	w.step(now.Add(3 * time.Second))
	w.handle(snoofer.Request{ID: "insta360.reset", Operation: "press"}, now.Add(3*time.Second))
	if last := cam.writes[len(cam.writes)-1]; last.selector != selBias || !bytes.Equal(last.data, []byte{0xFF, 0x7F, 0xFF, 0x7F}) {
		t.Fatal("tracking reset", last)
	}
}

func TestAbsentPreviewAndFailure(t *testing.T) {
	w, now := testWorker(true, nil)
	for _, c := range w.controls(*now) {
		if c.Kind != "connection" && c.Kind != "status" && (c.Available || c.Status != "No camera") {
			t.Fatal(c.ID, c.Available, c.Status)
		}
	}
	if r := control(t, w, *now, "insta360.app-camera"); r.Value != "No camera" || r.Connection.State != snoofer.ConnectionDisconnected {
		t.Fatal(r.Value, r.Connection)
	}

	cam := newFake()
	w, now = testWorker(false, cam)
	w.handle(snoofer.Request{ID: "insta360.privacy", Operation: "press"}, *now)
	if len(cam.writes) != 0 {
		t.Fatal("preview wrote to the camera")
	}
	if c := control(t, w, *now, "insta360.privacy"); c.Available || c.Status != "Preview" {
		t.Fatal(c.Available, c.Status)
	}

	cam = newFake()
	w, now = testWorker(true, cam)
	cam.failGet = errors.New("device removed")
	w.step(now.Add(3 * time.Second))
	if !cam.closed || w.dev != nil {
		t.Fatal("failed device kept open")
	}
	if r := control(t, w, now.Add(3*time.Second), "insta360.app-camera"); r.Connection.LastError != "device removed" {
		t.Fatal(r.Connection)
	}
}

func TestStopClosesCamera(t *testing.T) {
	cam := newFake()
	s := snoofer.Services{Controls: snoofer.NewControls(), Live: true}
	released := false
	i := start(context.Background(), s, func() (device, error) { return cam, nil }, func() error {
		released = true
		return nil
	})
	deadline := time.Now().Add(3 * time.Second)
	for len(s.Controls.Snapshot()) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := i.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if !cam.closed || !released || len(s.Controls.Snapshot()) != 0 {
		t.Fatal("stop", cam.closed, released, len(s.Controls.Snapshot()))
	}
}
