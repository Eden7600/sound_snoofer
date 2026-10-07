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
	if unsafe.Sizeof(nativeConfig{}) != 48 {
		t.Fatal("AECConfig V2 layout", unsafe.Sizeof(nativeConfig{}))
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
	if err := e.Configure(Config{Mic: [2]int{0, 1}, Reference: [8]int{0, 1, 2, 3, 4, 5, 6, 7}, Strength: Balanced}); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Config{
		{Mic: [2]int{-1, 1}, Reference: [8]int{0, 1, 2, 3, 4, 5, 6, 7}},
		{Mic: [2]int{0, 1}, Reference: [8]int{0, 1, 2, 3, 4, 5, 6, 128}},
		{Mic: [2]int{0, 1}, Reference: [8]int{0, 1, 2, 3, 4, 5, 6, 6}},
		{Mic: [2]int{0, 1}, Reference: [8]int{0, 1, 2, 3, 4, 5, 6, 7}, Strength: Strength(3)},
	} {
		if e.Configure(bad) == nil {
			t.Fatalf("accepted invalid config: %+v", bad)
		}
	}
	if _, err := e.dll.FindProc("AECConfigure"); err == nil {
		t.Fatal("unversioned config ABI still exported")
	}
	const samples = 480
	channels := make([][]float32, 16)
	for c := range channels {
		channels[c] = make([]float32, samples)
	}
	b := &audioBuffer{sr: 48000, samples: samples, inputs: 8, outputs: 8}
	for c := range 8 {
		b.read[c], b.write[c] = &channels[c][0], &channels[c+8][0]
	}
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
	// A stream that ends between the inserts latches a failure with its reason.
	e.inputInsert.Call(hook.Context, uintptr(unsafe.Pointer(b)))
	e.inputInsert.Call(hook.Context, uintptr(unsafe.Pointer(b)))
	if s, err := e.Stats(); err != nil || !s.Failed || s.Active || s.Reason != "missing output callback" {
		t.Fatal("fault not latched with its reason", s, err)
	}
	if err := e.Reset(); err != nil {
		t.Fatal(err)
	}
	for range 10 {
		e.inputInsert.Call(hook.Context, uintptr(unsafe.Pointer(b)))
		e.outputInsert.Call(hook.Context, uintptr(unsafe.Pointer(b)))
	}
	if s, err := e.Stats(); err != nil || s.Failed || !s.Active || s.Reason != "" {
		t.Fatal("reset did not resume processing", s, err)
	}
	if err := e.Configure(Config{Mic: [2]int{0, 1}, Reference: [8]int{0, 1, 2, 3, 4, 5, 6, 7}, Bypass: true}); err != nil {
		t.Fatal(err)
	}
	e.inputInsert.Call(hook.Context, uintptr(unsafe.Pointer(b)))
	if s, _ := e.Stats(); s.Active {
		t.Fatal("bypass still active", s)
	}
}

func TestNativeNeuralModels(t *testing.T) {
	path := os.Getenv("SNOOFER_NEURAL_AEC_DLL")
	if path == "" {
		t.Skip("set SNOOFER_NEURAL_AEC_DLL to the built neural engine")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"localvqe-v1.4-aec-200K-f32.gguf", "localvqe-v1.3-4.8M-f32.gguf", "localvqe-v1.4-aec-200K-f32.gguf"} {
		e, err := OpenNeural(path, filepath.Join(filepath.Dir(path), "models", name))
		if err != nil {
			t.Fatal(name, err)
		}
		if err := e.Configure(Config{Mic: [2]int{0, 1}, Reference: [8]int{0, 1, 2, 3, 4, 5, 6, 7}}); err != nil {
			t.Fatal(err)
		}
		if s, err := e.Stats(); err != nil || s.Active || s.Failed || s.ERLEKnown || s.DelayKnown {
			t.Fatal(s, err)
		}
		hook := e.Hook()
		if hook.Input == 0 || hook.Output == 0 || hook.Context == 0 {
			t.Fatal("missing neural hooks")
		}
		if err := e.Reset(); err != nil {
			t.Fatal(err)
		}
		if err := e.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
