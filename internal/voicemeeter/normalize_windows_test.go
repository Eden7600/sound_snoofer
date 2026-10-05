//go:build windows && amd64

package voicemeeter

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativePeakNormalization(t *testing.T) {
	if os.Getenv("SNOOFER_NORMALIZE_TEST") != "1" {
		t.Skip("set SNOOFER_NORMALIZE_TEST=1 with the native companion built")
	}
	dll, err := filepath.Abs("../../bin/snoofer-soundboard.dll")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for _, silent := range []bool{false, true} {
		input := filepath.Join(root, "input.wav")
		output := filepath.Join(root, "output.wav")
		data := make([]byte, 44+4800*2)
		copy(data, "RIFF")
		binary.LittleEndian.PutUint32(data[4:], uint32(len(data)-8))
		copy(data[8:], "WAVEfmt ")
		binary.LittleEndian.PutUint32(data[16:], 16)
		binary.LittleEndian.PutUint16(data[20:], 1)
		binary.LittleEndian.PutUint16(data[22:], 1)
		binary.LittleEndian.PutUint32(data[24:], 48000)
		binary.LittleEndian.PutUint32(data[28:], 96000)
		binary.LittleEndian.PutUint16(data[32:], 2)
		binary.LittleEndian.PutUint16(data[34:], 16)
		copy(data[36:], "data")
		binary.LittleEndian.PutUint32(data[40:], uint32(len(data)-44))
		if !silent {
			for n := 0; n < 4800; n++ {
				v := int16(1000 * math.Sin(2*math.Pi*float64(n)/48))
				binary.LittleEndian.PutUint16(data[44+n*2:], uint16(v))
			}
		}
		if err := os.WriteFile(input, data, 0600); err != nil {
			t.Fatal(err)
		}
		if err := NormalizeClip(context.Background(), dll, input, output); err != nil {
			t.Fatal(err)
		}
		normalized, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		peak := pcmPeak(t, normalized)
		if silent && peak != 0 {
			t.Fatal("silence amplified")
		}
		if !silent && math.Abs(float64(peak)-32768*math.Pow(10, -1.0/20)) > 1 {
			t.Fatal("wrong peak", peak)
		}
		original, err := os.ReadFile(input)
		if err != nil || sha256.Sum256(original) != sha256.Sum256(data) {
			t.Fatal("source changed")
		}
	}
	if source := os.Getenv("SNOOFER_NORMALIZE_TEST_MP3"); source != "" {
		original, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		output := filepath.Join(root, "mp3.wav")
		if err := NormalizeClip(context.Background(), dll, source, output); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		peak := pcmPeak(t, data)
		if math.Abs(float64(peak)-32768*math.Pow(10, -1.0/20)) > 1 {
			t.Fatal("MP3 peak incorrect", peak)
		}
		after, err := os.ReadFile(source)
		if err != nil || sha256.Sum256(original) != sha256.Sum256(after) {
			t.Fatal("MP3 changed")
		}
		t.Logf("MP3 normalized peak: %.4f dBFS", 20*math.Log10(float64(peak)/32768))
		runtime.LockOSThread()
		player, err := OpenClipPlayer(dll, "DirectSound: Voicemeeter VAIO3 Input (VB-Audio Voicemeeter VAIO)")
		if err == nil {
			err = errors.Join(player.Play(output, true), player.Close())
		}
		runtime.UnlockOSThread()
		if err != nil {
			t.Fatal("normalized WAV playback:", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := NormalizeClip(ctx, dll, "missing.mp3", filepath.Join(root, "cancelled.wav")); !errors.Is(err, context.Canceled) {
		t.Fatal("cancel ignored", err)
	}
	if err := NormalizeClip(context.Background(), dll, filepath.Join(root, "missing.mp3"), filepath.Join(root, "missing.wav")); err == nil {
		t.Fatal("missing input accepted")
	}
}
func pcmPeak(t *testing.T, data []byte) int {
	t.Helper()
	if len(data) < 46 || string(data[:4]) != "RIFF" || string(data[36:40]) != "data" {
		t.Fatal("invalid output WAV")
	}
	peak := 0
	for n := 44; n+1 < len(data); n += 2 {
		value := int(int16(binary.LittleEndian.Uint16(data[n:])))
		if value < 0 {
			value = -value
		}
		if value > peak {
			peak = value
		}
	}
	return peak
}
