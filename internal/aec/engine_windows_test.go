//go:build windows

package aec

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"unsafe"
)

// audioBuffer mirrors Voicemeeter's AudioBuffer on Windows x64.
type audioBuffer struct {
	sr, samples, inputs, outputs int32
	read, write                  [128]*float32
}

// Opt-in: SNOOFER_AEC_DLL names a built snoofer-aec.dll. Drives the insert
// stages directly to check the Go mirrors of the C ABI.
func TestNativeEngine(t *testing.T) {
	path := os.Getenv("SNOOFER_AEC_DLL")
	if path == "" {
		t.Skip("set SNOOFER_AEC_DLL to the built engine")
	}
	if unsafe.Sizeof(audioBuffer{}) != 2064 {
		t.Fatal("AudioBuffer layout", unsafe.Sizeof(audioBuffer{}))
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	e, err := Open(abs)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := e.Close(); err != nil {
			t.Error(err)
		}
	}()
	if s, err := e.Stats(); err != nil || s.Active || s.ERLEKnown || s.DelayKnown {
		t.Fatal(s, err)
	}
	if err := e.Configure(Config{Mic: [2]int{0, 1}, Reference: [2]int{0, 1}, Strength: Balanced}); err != nil {
		t.Fatal(err)
	}
	const samples = 480
	channels := make([][]float32, 4)
	for c := range channels {
		channels[c] = make([]float32, samples)
	}
	b := &audioBuffer{sr: 48000, samples: samples, inputs: 2, outputs: 2}
	b.read[0], b.read[1] = &channels[0][0], &channels[1][0]
	b.write[0], b.write[1] = &channels[2][0], &channels[3][0]
	hook := e.Hook()
	for cycle := 0; cycle < 100; cycle++ {
		for i := range samples {
			v := float32(0.1 * math.Sin(float64(cycle*samples+i)*0.05))
			channels[0][i], channels[1][i] = v, v
		}
		e.inputInsert.Call(hook.Context, uintptr(unsafe.Pointer(b)))
		e.outputInsert.Call(hook.Context, uintptr(unsafe.Pointer(b)))
	}
	s, err := e.Stats()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", s)
	if !s.Active || s.SampleRate != 48000 || s.Frames < 98 || s.Failed {
		t.Fatal(s)
	}
	if err := e.Configure(Config{Mic: [2]int{0, 1}, Reference: [2]int{0, 1}, Bypass: true}); err != nil {
		t.Fatal(err)
	}
	e.inputInsert.Call(hook.Context, uintptr(unsafe.Pointer(b)))
	if s, _ := e.Stats(); s.Active {
		t.Fatal("bypass still active", s)
	}
}
