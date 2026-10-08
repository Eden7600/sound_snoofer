// Package aec wraps snoofer-aec.dll, WebRTC AEC3 behind a small C ABI whose
// insert stages run inside the Voicemeeter audio callback.
package aec

// Strength selects how hard the echo suppressor works.
type Strength int

const (
	Strong   Strength = 0 // AEC3's default suppressor.
	Balanced Strength = 1
	Gentle   Strength = 2 // Least suppression; keeps the most local speech.
)

// Config is the engine's requested setup. Channel indexes refer to the
// Voicemeeter insert buffers; -1 marks an unused channel.
type Config struct {
	Mic       [2]int // Input-insert channels of the mic strip.
	Reference [8]int // Output-insert channels of the speaker bus.
	Strength  Strength
	Bypass    bool // Pass the mic through untouched.
}

// Stats is the engine's latest report from the audio thread.
type Stats struct {
	TimingKnown               bool
	WorkerPeakMs, QueuePeakMs float64 // Peaks since engine load, including warm-up.
	Gaps, Underruns           int32
	LatencyMs                 int  // Processing delay, including framing; zero when unknown.
	Active                    bool // Processing with a supported rate and targets.
	SampleRate                int  // As last seen from Voicemeeter; 0 before any audio.
	ERLEKnown                 bool
	ERLE                      float64 // Echo return loss enhancement in dB.
	DelayKnown                bool
	DelayMs                   int    // Estimated echo delay.
	Frames                    uint32 // 10 ms frames processed.
	Failed                    bool   // An engine error switched to pass-through.
	Reason                    string // Why the engine failed; empty when it has not.
}

// Supported reports whether the engine processes at a sample rate.
func Supported(rate int) bool {
	return rate == 48000 || rate == 32000 || rate == 16000
}

// reasons maps AECReadFailure codes to status text.
var reasons = map[int32]string{
	9:  "neural audio queue overflow",
	10: "non-finite audio",
	11: "neural callback exceeds 2048 samples",
	1:  "missing output callback",
	2:  "missing microphone channel",
	3:  "capture failed",
	4:  "capture underflow",
	5:  "unpaired output callback",
	6:  "missing reference channel",
	7:  "reference failed",
	8:  "unexpected exception",
}
