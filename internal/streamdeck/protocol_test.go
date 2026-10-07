package streamdeck

import (
	"encoding/binary"

	"testing"
)

func report(cmd byte, p []byte) []byte { b := []byte{1, cmd, byte(len(p)), 0}; return append(b, p...) }
func TestKeyEdgesReconnectAndMalformedReports(t *testing.T) {
	d := Decoder{}
	keys := make([]byte, Keys)
	keys[2] = 1
	if e, err := d.Decode(report(0, keys)); err != nil || len(e) != 0 {
		t.Fatal("initial held key fired")
	}
	keys[2] = 0
	d.Decode(report(0, keys))
	keys[2] = 1
	e, err := d.Decode(report(0, keys))
	if err != nil || len(e) != 1 || e[0].Key != 2 {
		t.Fatal(e, err)
	}
	e, _ = d.Decode(report(0, keys))
	if len(e) != 0 {
		t.Fatal("held repeat")
	}
	if _, err = d.Decode([]byte{1, 0, 36, 0, 1}); err == nil {
		t.Fatal("truncation accepted")
	}
}
func TestSignedEncoderAndImageFrames(t *testing.T) {
	d := Decoder{}
	e, err := d.Decode(report(3, []byte{1, 255, 2, 0, 0, 0, 0}))
	if err != nil || len(e) != 2 || e[0].Delta != -1 {
		t.Fatal(e, err)
	}
	frames, err := ImageReports(2, false, make([]byte, 100), 64)
	if err != nil || len(frames) != 2 || frames[1][3] != 1 || binary.LittleEndian.Uint16(frames[1][6:8]) != 1 {
		t.Fatal(frames, err)
	}
}

func TestKeyReleases(t *testing.T) {
	d := Decoder{}
	keys := make([]byte, Keys)
	d.Decode(report(0, keys))
	keys[5] = 1
	if e, _ := d.Decode(report(0, keys)); len(e) != 1 || !e[0].Press || e[0].Release {
		t.Fatal("press", e)
	}
	keys[5] = 0
	e, err := d.Decode(report(0, keys))
	if err != nil || len(e) != 1 || e[0].Key != 5 || e[0].Encoder != -1 || !e[0].Release || e[0].Press {
		t.Fatal("release", e, err)
	}
	if e, _ := d.Decode(report(0, keys)); len(e) != 0 {
		t.Fatal("repeated release", e)
	}
}
