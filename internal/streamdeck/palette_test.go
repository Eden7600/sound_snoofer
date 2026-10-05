package streamdeck

import (
	"image/color"
	"strings"
	"testing"
)

func TestSemanticAccentPrecedence(t *testing.T) {
	tests := []struct {
		value, icon  string
		on, fallback bool
		want         color.RGBA
	}{
		{"N/A", "mic-mute-muted", true, true, neutralColor},
		{"Error", "mic-mute-muted", true, true, criticalColor},
		{"Wait", "mic-mute-muted", true, false, attentionColor},
		{"On", "mic-mute-muted", true, false, criticalColor},
		{"Off", "mic-stack-off", false, false, neutralColor},
		{"Ready", "record-toggle", false, false, neutralColor},
		{"Recording", "record-stop", true, false, criticalColor},
		{"Direct", "mode-direct", false, false, activeColor},
		{"Element", "mode-element", false, false, activeColor},
		{"Pre", "tap-pre", false, false, neutralColor},
		{"Pre", "monitor", false, false, activeColor},
		{"On", "record-mic", true, true, attentionColor},
	}
	for _, tt := range tests {
		if got := keyAccent(strings.ToUpper(tt.value), tt.icon, tt.on, tt.fallback); got != tt.want {
			t.Errorf("%+v: got %v", tt, got)
		}
	}
}
