//go:build windows

package aec

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"

	"sound-snoofer/internal/voicemeeter"
)

// Engine owns one native echo canceller. Configure, Stats and Close may run
// on any one goroutine at a time. Close frees the engine and the DLL, so it
// may run only once the audio callback no longer calls Hook's stages.
type Engine struct {
	dll                       *windows.DLL
	handle                    uintptr
	destroy, configure, stats *windows.Proc
	inputInsert, outputInsert *windows.Proc
}

// nativeConfig mirrors AECConfig: six 32-bit ints.
type nativeConfig struct {
	mic, reference   [2]int32
	strength, bypass int32
}

// nativeStats mirrors AECStats: six 32-bit fields.
type nativeStats struct {
	active, sampleRate, erleCentiDB, delayMs int32
	frames                                   uint32
	failed                                   int32
}

func result(name string, code uintptr) error {
	if int32(code) < 0 {
		return fmt.Errorf("%s returned 0x%08X", name, uint32(code))
	}
	return nil
}

// Open loads the DLL at path and creates an engine.
func Open(path string) (*Engine, error) {
	dll, err := windows.LoadDLL(path)
	if err != nil {
		return nil, err
	}
	e := &Engine{dll: dll}
	var create *windows.Proc
	procs := map[string]**windows.Proc{
		"AECCreate": &create, "AECDestroy": &e.destroy, "AECConfigure": &e.configure, "AECReadStats": &e.stats,
		"AECInputInsert": &e.inputInsert, "AECOutputInsert": &e.outputInsert,
	}
	for name, target := range procs {
		if *target, err = dll.FindProc(name); err != nil {
			return nil, errors.Join(err, dll.Release())
		}
	}
	code, _, _ := create.Call(uintptr(unsafe.Pointer(&e.handle)))
	if err := result("AECCreate", code); err != nil {
		return nil, errors.Join(err, dll.Release())
	}
	return e, nil
}

// Configure publishes a new setup to the audio thread.
func (e *Engine) Configure(c Config) error {
	native := nativeConfig{
		mic:       [2]int32{int32(c.Mic[0]), int32(c.Mic[1])},
		reference: [2]int32{int32(c.Reference[0]), int32(c.Reference[1])},
		strength:  int32(c.Strength),
	}
	if c.Bypass {
		native.bypass = 1
	}
	code, _, _ := e.configure.Call(e.handle, uintptr(unsafe.Pointer(&native)))
	return result("AECConfigure", code)
}

// Stats reads the audio thread's latest report.
func (e *Engine) Stats() (Stats, error) {
	var native nativeStats
	code, _, _ := e.stats.Call(e.handle, uintptr(unsafe.Pointer(&native)))
	if err := result("AECReadStats", code); err != nil {
		return Stats{}, err
	}
	return Stats{
		Active:     native.active != 0,
		SampleRate: int(native.sampleRate),
		ERLEKnown:  native.erleCentiDB >= 0,
		ERLE:       float64(native.erleCentiDB) / 100,
		DelayKnown: native.delayMs >= 0,
		DelayMs:    int(native.delayMs),
		Frames:     native.frames,
		Failed:     native.failed != 0,
	}, nil
}

// Hook returns the insert stages for the monitor's audio callback.
func (e *Engine) Hook() voicemeeter.InsertHook {
	return voicemeeter.InsertHook{Input: e.inputInsert.Addr(), Output: e.outputInsert.Addr(), Context: e.handle}
}

// Close destroys the engine and releases the DLL.
func (e *Engine) Close() error {
	code, _, _ := e.destroy.Call(e.handle)
	e.handle = 0
	return errors.Join(result("AECDestroy", code), e.dll.Release())
}
