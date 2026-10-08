package audio

import (
	"testing"

	"sound-snoofer/internal/config"
	"sound-snoofer/snoofer"
)

func TestGainStepSetting(t *testing.T) {
	for _, tc := range []struct {
		step *float64
		ok   bool
		want float32
	}{{nil, true, 1}, {ptr(0.5), true, 0.5}, {ptr(6), true, 6}, {ptr(0), false, 0}, {ptr(-1), false, 0}, {ptr(6.5), false, 0}} {
		settings := Settings{Config: config.DefaultBytes(), StatePath: "audio", GainStepDB: tc.step}
		err := validateSettings(snoofer.MarshalSettings(settings))
		if (err == nil) != tc.ok {
			t.Fatal(tc.step, err)
		}
		if tc.ok && settings.gainStep() != tc.want {
			t.Fatal(settings.gainStep(), tc.want)
		}
	}
}

func ptr(v float64) *float64 { return &v }
