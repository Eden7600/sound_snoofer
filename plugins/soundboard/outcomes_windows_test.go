//go:build windows && amd64

package soundboard

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"sound-snoofer/plugins/audio"
	"sound-snoofer/snoofer"
)

type outcomeAudio struct{}

func (a *outcomeAudio) SetSoundboardRoutes(*audio.SoundboardRoutes) error { return nil }
func (a *outcomeAudio) SoundboardReady() error                            { return nil }
func (a *outcomeAudio) SoundboardVolume() (snoofer.Control, func(context.Context, snoofer.Request) error) {
	return snoofer.Control{ID: "soundboard.volume", Kind: "numeric"}, func(context.Context, snoofer.Request) error { return nil }
}

func outcomePCM(path string) error {
	data := make([]byte, 46)
	copy(data, "RIFF")
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[4:], uint32(len(data)-8))
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], 48000)
	binary.LittleEndian.PutUint16(data[34:], 16)
	binary.LittleEndian.PutUint32(data[40:], uint32(len(data)-44))
	copy(data[36:], "data")
	return os.WriteFile(path, data, 0600)
}

func soundControl(t *testing.T, registry *snoofer.Controls, id string) snoofer.Control {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		for _, c := range registry.Snapshot() {
			if c.ID == id || (id == "clip" && c.Collection == "soundboard.clips") {
				return c
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("missing soundboard control", id)
	return snoofer.Control{}
}

func startSoundboardOutcome(t *testing.T, decode func(context.Context, string, string, string) error, open func(string, string) (clipPlayer, error), save func(string, json.RawMessage, json.RawMessage) error) (*snoofer.Controls, context.CancelFunc, string) {
	t.Helper()
	folder := t.TempDir()
	clipPath := filepath.Join(folder, "sound.mp3")
	if err := os.WriteFile(clipPath, []byte("clip"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	registry := snoofer.NewControls()
	settings := Settings{Folder: folder, Renderer: "DirectSound: Voicemeeter VAIO3 Input", Microphone: true}
	instance, err := startWithPlayback(ctx, snoofer.Services{Path: filepath.Join(folder, "settings.json"), Controls: registry, Live: true, SaveSettings: save}, snoofer.MarshalSettings(settings), &outcomeAudio{}, open, decode)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		stop, release := context.WithTimeout(context.Background(), time.Second)
		defer release()
		if err := instance.Stop(stop); err != nil {
			t.Error(err)
		}
	})
	soundControl(t, registry, "clip")
	return registry, cancel, clipPath
}

func sendSound(t *testing.T, registry *snoofer.Controls, id string) <-chan snoofer.Outcome {
	t.Helper()
	c := soundControl(t, registry, id)
	results := make(chan snoofer.Outcome, 2)
	if err := registry.DispatchWithResult(context.Background(), snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "press"}, func(r snoofer.Outcome) { results <- r }); err != nil {
		t.Fatal(err)
	}
	return results
}

func TestSoundboardCompletionWaitsForPreparationAndPlay(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	registry, _, _ := startSoundboardOutcome(t, func(ctx context.Context, _, _, destination string) error {
		close(entered)
		select {
		case <-release:
			return outcomePCM(destination)
		case <-ctx.Done():
			return ctx.Err()
		}
	}, func(string, string) (clipPlayer, error) { return &fakePlayer{}, nil }, nil)
	defer close(release)
	results := sendSound(t, registry, "clip")
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("preparation did not start")
	}
	select {
	case result := <-results:
		t.Fatalf("queue acceptance claimed playback: %+v", result)
	default:
	}
	release <- struct{}{}
	select {
	case result := <-results:
		if result.Status != snoofer.Processed {
			t.Fatal(result)
		}
	case <-time.After(time.Second):
		t.Fatal("missing playback result")
	}
}

type outcomePlayer struct {
	plays atomic.Int32
	stops atomic.Int32
	stop  func() error
}

func (p *outcomePlayer) Play(string, bool) error { p.plays.Add(1); return nil }
func (p *outcomePlayer) Stop() error {
	p.stops.Add(1)
	if p.stop != nil {
		return p.stop()
	}
	return nil
}
func (p *outcomePlayer) Poll() (bool, error) { return true, nil }
func (p *outcomePlayer) Close() error        { return nil }

func expectSound(t *testing.T, results <-chan snoofer.Outcome, status snoofer.OutcomeStatus) snoofer.Outcome {
	t.Helper()
	select {
	case result := <-results:
		if result.Status != status {
			t.Fatalf("got %+v, want %s", result, status)
		}
		return result
	case <-time.After(time.Second):
		t.Fatal("missing soundboard result")
	}
	return snoofer.Outcome{}
}

func waitSound(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("soundboard worker did not reach checkpoint")
	}
}

func immediatePCM(_ context.Context, _, _, destination string) error { return outcomePCM(destination) }

func TestSoundboardReplacementSettlesOldPreparationWithoutLatePlay(t *testing.T) {
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	player := &outcomePlayer{}
	registry, cancel, _ := startSoundboardOutcome(t, func(_ context.Context, _, _, destination string) error {
		if calls.Add(1) == 1 {
			close(entered)
			<-release
			defer close(finished)
		}
		return outcomePCM(destination)
	}, func(string, string) (clipPlayer, error) { return player, nil }, nil)
	defer close(release)
	first := sendSound(t, registry, "clip")
	waitSound(t, entered)
	second := sendSound(t, registry, "clip")
	if result := expectSound(t, first, snoofer.Rejected); result.Detail != "Cancelled before playback" {
		t.Fatal(result)
	}
	if calls.Load() != 1 {
		t.Fatal("decoders overlapped")
	}
	release <- struct{}{}
	waitSound(t, finished)
	expectSound(t, second, snoofer.Processed)
	cancel()
	if player.plays.Load() != 1 {
		t.Fatal("obsolete preparation played", player.plays.Load())
	}
	select {
	case result := <-first:
		t.Fatal("late duplicate", result)
	default:
	}
}

func TestSoundboardStopAndShutdownDuringPreparation(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		t.Run(map[bool]string{false: "stop", true: "shutdown"}[shutdown], func(t *testing.T) {
			entered, finished := make(chan struct{}), make(chan struct{})
			var opened atomic.Int32
			registry, cancel, _ := startSoundboardOutcome(t, func(ctx context.Context, _, _, _ string) error {
				close(entered)
				<-ctx.Done()
				close(finished)
				return ctx.Err()
			}, func(string, string) (clipPlayer, error) { opened.Add(1); return &outcomePlayer{}, nil }, nil)
			play := sendSound(t, registry, "clip")
			waitSound(t, entered)
			if shutdown {
				cancel()
				expectSound(t, play, snoofer.Unknown)
			} else {
				stop := sendSound(t, registry, "soundboard.stop")
				expectSound(t, play, snoofer.Rejected)
				expectSound(t, stop, snoofer.Processed)
			}
			waitSound(t, finished)
			if opened.Load() != 0 {
				t.Fatal("cancelled preparation opened playback")
			}
		})
	}
}

func TestSoundboardPreparationAndNativeFailures(t *testing.T) {
	for _, stage := range []string{"prepare", "open", "play"} {
		t.Run(stage, func(t *testing.T) {
			failure := errors.New(stage + " failed")
			registry, _, _ := startSoundboardOutcome(t, func(ctx context.Context, library, source, destination string) error {
				if stage == "prepare" {
					return failure
				}
				return immediatePCM(ctx, library, source, destination)
			}, func(string, string) (clipPlayer, error) {
				if stage == "open" {
					return nil, failure
				}
				p := &fakePlayer{}
				if stage == "play" {
					p.playErr = failure
				}
				return p, nil
			}, nil)
			result := expectSound(t, sendSound(t, registry, "clip"), snoofer.Failed)
			if !strings.Contains(result.Detail, failure.Error()) {
				t.Fatal(result)
			}
		})
	}
}

func TestSoundboardStopWaitsForNativeResult(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	player := &outcomePlayer{stop: func() error { close(entered); <-release; return errors.New("native stop failed") }}
	registry, _, _ := startSoundboardOutcome(t, immediatePCM, func(string, string) (clipPlayer, error) { return player, nil }, nil)
	defer close(release)
	expectSound(t, sendSound(t, registry, "clip"), snoofer.Processed)
	result := sendSound(t, registry, "soundboard.stop")
	waitSound(t, entered)
	select {
	case r := <-result:
		t.Fatal("stop completed before native call", r)
	default:
	}
	release <- struct{}{}
	if r := expectSound(t, result, snoofer.Failed); r.Detail != "native stop failed" {
		t.Fatal(r)
	}
}

func TestSoundboardOverlapRejectsQueuedOldIntentAndFullQueue(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var saves atomic.Int32
	registry, _, _ := startSoundboardOutcome(t, immediatePCM, func(string, string) (clipPlayer, error) { return &outcomePlayer{}, nil }, func(_ string, _, data json.RawMessage) error {
		saves.Add(1)
		var settings Settings
		if err := json.Unmarshal(data, &settings); err != nil || !settings.Overlap {
			t.Errorf("wrong explicit overlap intent: %s %v", data, err)
		}
		close(entered)
		<-release
		return nil
	})
	defer close(release)
	first := sendSound(t, registry, "soundboard.overlap")
	waitSound(t, entered)
	second := sendSound(t, registry, "soundboard.overlap")
	c := soundControl(t, registry, "soundboard.stop")
	full := make(chan snoofer.Outcome, 2)
	err := registry.DispatchWithResult(context.Background(), snoofer.Request{ID: c.ID, Revision: c.Revision, Operation: "press"}, func(r snoofer.Outcome) { full <- r })
	if err == nil {
		t.Fatal("full queue accepted action")
	}
	expectSound(t, full, snoofer.Rejected)
	select {
	case result := <-first:
		t.Fatal("save completed early", result)
	default:
	}
	release <- struct{}{}
	expectSound(t, first, snoofer.Processed)
	expectSound(t, second, snoofer.Rejected)
	if saves.Load() != 1 {
		t.Fatal("queued toggle inverted newer settings")
	}
}

func TestSoundboardSaveFailureAndQueuedShutdown(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		t.Run(map[bool]string{false: "save failure", true: "shutdown"}[shutdown], func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			registry, cancel, _ := startSoundboardOutcome(t, immediatePCM, func(string, string) (clipPlayer, error) {
				t.Error("queued clip played after shutdown")
				return &outcomePlayer{}, nil
			}, func(string, json.RawMessage, json.RawMessage) error {
				close(entered)
				<-release
				return errors.New("save failed")
			})
			defer close(release)
			first := sendSound(t, registry, "soundboard.overlap")
			waitSound(t, entered)
			if shutdown {
				second := sendSound(t, registry, "clip")
				cancel()
				expectSound(t, first, snoofer.Unknown)
				expectSound(t, second, snoofer.Unknown)
				release <- struct{}{}
			} else {
				release <- struct{}{}
				if r := expectSound(t, first, snoofer.Failed); r.Detail != "save failed" {
					t.Fatal(r)
				}
			}
		})
	}
}

func TestSoundboardChangedClipRejectedBeforeStoppingCurrentVoice(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	player := &outcomePlayer{}
	registry, _, clipPath := startSoundboardOutcome(t, immediatePCM, func(string, string) (clipPlayer, error) { return player, nil }, func(string, json.RawMessage, json.RawMessage) error {
		close(entered)
		<-release
		return errors.New("keep overlap off")
	})
	defer close(release)
	expectSound(t, sendSound(t, registry, "clip"), snoofer.Processed)
	save := sendSound(t, registry, "soundboard.overlap")
	waitSound(t, entered)
	play := sendSound(t, registry, "clip")
	if err := os.WriteFile(clipPath, []byte("changed clip"), 0600); err != nil {
		t.Fatal(err)
	}
	release <- struct{}{}
	expectSound(t, save, snoofer.Failed)
	expectSound(t, play, snoofer.Rejected)
	if player.stops.Load() != 0 || player.plays.Load() != 1 {
		t.Fatal("stale clip changed current playback")
	}
}
