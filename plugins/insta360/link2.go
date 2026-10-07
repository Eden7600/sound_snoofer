package insta360

import (
	"encoding/binary"
	"fmt"

	"sound-snoofer/internal/camera"
)

// Link 2 extension units (UVC property sets) and selectors, from the official
// Insta360 Stream Deck plugin's camera controller. See the meetings design.
var (
	xu1 = camera.GUID{Data1: 0xFAF1672D, Data2: 0xB71B, Data3: 0x4793, Data4: [8]byte{0x8C, 0x91, 0x7B, 0x1C, 0x9B, 0x7F, 0x95, 0xF8}}
	xu2 = camera.GUID{Data1: 0xE307E649, Data2: 0x4618, Data3: 0xA3FF, Data4: [8]byte{0x82, 0xFC, 0x2D, 0x8B, 0x5F, 0x21, 0x67, 0x73}}
)

const (
	devicePath = "vid_2e1a&pid_4c04" // Insta360 Link 2.

	selVideoMode   = 0x02 // XU1: status packet (GET) and video mode (SET).
	selComposition = 0x13 // XU1: framing style.
	selBias        = 0x18 // XU1: composition bias, two int16.
	selPanTilt     = 0x1A // XU1: absolute pan and tilt, two int32.
	selExtended    = 0x1B // XU1: extended-function bits, uint16.
	selPrivacy     = 0x0F // XU2: privacy, one byte.

	extendedAIZoom = 1 << 0 // Framing styles need AI zoom.
)

// Video modes used by tracking.
const (
	modeNormal          = 0
	modeAutoComposition = 1 // Single person.
	modeAutoFraming     = 7 // Group.
)

// tracking maps option values to video modes and back.
var (
	trackingModes = map[string]byte{"off": modeNormal, "single": modeAutoComposition, "group": modeAutoFraming}
	framingStyles = map[string]byte{"head": 1, "half": 2, "full": 3}
)

// status is the parsed XU1 0x02 packet. Firmware varies its length, so each
// field records whether the packet reached it.
type status struct {
	Mode, ModeState       byte
	ModeKnown             bool
	Privacy, PrivacyKnown bool
}

// Offsets in the status packet.
const (
	offsetFlags = 0x36 // uint16; bit 0 is privacy.
)

func parseStatus(p []byte) status {
	var s status
	if len(p) >= 2 {
		s.Mode, s.ModeState, s.ModeKnown = p[0], p[1], true
	}
	if len(p) >= offsetFlags+2 {
		flags := binary.LittleEndian.Uint16(p[offsetFlags:])
		s.Privacy, s.PrivacyKnown = flags&1 != 0, true
	}
	return s
}

// tracking names a video mode; modes outside tracking (whiteboard, desk view)
// are reported as they are but are not tracking options.
func trackingName(mode byte) string {
	for name, m := range trackingModes {
		if m == mode {
			return name
		}
	}
	return fmt.Sprintf("mode %d", mode)
}

func framingName(style byte) string {
	for name, s := range framingStyles {
		if s == style {
			return name
		}
	}
	return "none"
}

// stateWord describes what tracking is doing.
func stateWord(s status) string {
	switch {
	case !s.ModeKnown:
		return "Unknown"
	case s.Mode == modeNormal:
		return "Idle"
	}
	switch s.ModeState {
	case 1:
		return "Detecting"
	case 2:
		return "Working"
	case 3:
		return "Lost"
	}
	return "Idle"
}

// videoModePacket sets only the mode. The pose fields carry the values the
// official plugin sends to leave the gimbal and zoom where they are.
func videoModePacket(mode byte) []byte {
	p := make([]byte, 0x38)
	p[0] = mode
	for _, offset := range []int{0x26, 0x2A, 0x2E} { // pan, tilt, roll
		binary.LittleEndian.PutUint32(p[offset:], 3610)
	}
	binary.LittleEndian.PutUint16(p[0x32:], 0) // zoom
	var pitch int16 = -450
	binary.LittleEndian.PutUint16(p[0x34:], uint16(pitch))
	return p
}

// centrePanTilt is an absolute pan and tilt of zero.
func centrePanTilt() []byte {
	return make([]byte, 8)
}

// centreBias re-centres the composition while tracking.
func centreBias() []byte {
	return []byte{0xFF, 0x7F, 0xFF, 0x7F}
}

// withAIZoom sets the AI-zoom bit in the extended-function word, reporting
// whether it changed.
func withAIZoom(word []byte) ([]byte, bool, error) {
	if len(word) < 2 {
		return nil, false, fmt.Errorf("extended functions: %d bytes", len(word))
	}
	bits := binary.LittleEndian.Uint16(word)
	if bits&extendedAIZoom != 0 {
		return word, false, nil
	}
	out := append([]byte(nil), word...)
	binary.LittleEndian.PutUint16(out, bits|extendedAIZoom)
	return out, true, nil
}
