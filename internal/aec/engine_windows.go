//go:build windows

package aec

import (
	"errors"
	"fmt"
	"runtime"
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
	readFailure, resetFailure *windows.Proc
	inputInsert, outputInsert *windows.Proc
	latency                   *windows.Proc
}

// nativeConfig mirrors AECConfig: twelve 32-bit ints.
type nativeConfig struct {
	mic              [2]int32
	reference        [8]int32
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
func Open(path string) (*Engine, error) { return open(path, "", false) }

// OpenNeural loads the neural companion and its selected model.
func OpenNeural(path, model string) (*Engine, error) { return open(path, model, false) }

// OpenFullband adds echo-controlled upper frequencies to the neural voice model.
func OpenFullband(path, model string) (*Engine, error) { return open(path, model, true) }

func open(path, model string, fullband bool) (*Engine, error) {
	handle, err := windows.LoadLibraryEx(path, 0, windows.LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|windows.LOAD_LIBRARY_SEARCH_DEFAULT_DIRS)
	dll := &windows.DLL{Name: path, Handle: handle}
	if err != nil {
		return nil, err
	}
	e := &Engine{dll: dll}
	var create *windows.Proc
	createName := "AECCreate"
	if model != "" {
		createName = "AECNeuralCreate"
		if fullband {
			createName = "AECNeuralFullbandCreate"
		}
	}
	procs := map[string]**windows.Proc{
		createName: &create, "AECDestroy": &e.destroy, "AECConfigureV2": &e.configure, "AECReadStats": &e.stats,
		"AECReadFailure": &e.readFailure, "AECResetFailure": &e.resetFailure,
		"AECInputInsert": &e.inputInsert, "AECOutputInsert": &e.outputInsert,
	}
	for name, target := range procs {
		if *target, err = dll.FindProc(name); err != nil {
			return nil, errors.Join(err, dll.Release())
		}
	}
	if model != "" {
		e.latency, err = dll.FindProc("AECReadLatency")
		if err != nil {
			return nil, errors.Join(err, dll.Release())
		}
	}
	args := []uintptr{uintptr(unsafe.Pointer(&e.handle))}
	var modelPath *byte
	if model != "" {
		modelPath, err = windows.BytePtrFromString(model)
		if err != nil {
			return nil, errors.Join(err, dll.Release())
		}
		args = append(args, uintptr(unsafe.Pointer(modelPath)))
	}
	code, _, _ := create.Call(args...)
	runtime.KeepAlive(modelPath)
	if err := result("AECCreate", code); err != nil {
		return nil, errors.Join(err, dll.Release())
	}
	return e, nil
}

// Configure publishes a new setup to the audio thread.
func (e *Engine) Configure(c Config) error {
	native := nativeConfig{
		mic:      [2]int32{int32(c.Mic[0]), int32(c.Mic[1])},
		strength: int32(c.Strength),
	}
	for i, channel := range c.Reference {
		native.reference[i] = int32(channel)
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
	stats := Stats{
		Active:     native.active != 0,
		SampleRate: int(native.sampleRate),
		ERLEKnown:  native.erleCentiDB >= 0,
		ERLE:       float64(native.erleCentiDB) / 100,
		DelayKnown: native.delayMs >= 0,
		DelayMs:    int(native.delayMs),
		Frames:     native.frames,
		Failed:     native.failed != 0,
	}
	if e.latency != nil {
		var latency int32
		code, _, _ := e.latency.Call(e.handle, uintptr(unsafe.Pointer(&latency)))
		if err := result("AECReadLatency", code); err != nil {
			return stats, err
		}
		stats.LatencyMs = int(latency)
	}
	if !stats.Failed {
		return stats, nil
	}
	var reason int32
	code, _, _ = e.readFailure.Call(e.handle, uintptr(unsafe.Pointer(&reason)))
	if err := result("AECReadFailure", code); err != nil {
		return stats, err
	}
	stats.Reason = reasons[reason]
	return stats, nil
}

// Reset clears a latched failure. The next input insert rebuilds the engine
// and its framing before processing resumes.
func (e *Engine) Reset() error {
	code, _, _ := e.resetFailure.Call(e.handle)
	return result("AECResetFailure", code)
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
