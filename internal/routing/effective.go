package routing

import (
	"sound-snoofer/internal/config"
	"sound-snoofer/internal/model"
)

// ResolveIntent derives achievable choices without changing saved preferences.
func ResolveIntent(preferred *config.Intent, s model.Snapshot) *config.Intent {
	i := preferred.Clone()
	if i == nil {
		return nil
	}
	if i.Mode == "element" && !s.ElementRunning() {
		i.Mode = "direct"
	}
	if i.Mode == "direct" {
		if i.Monitor == "post" {
			i.Monitor = "pre"
		}
		if i.Recording != nil {
			if i.Recording.MicTap == "post" {
				i.Recording.MicTap = "pre"
			}
			i.Recording.ToVST = false
		}
	}

	if i.Recording != nil && i.Recording.ToVST && s.Recorder != nil {
		recording := s.Recorder.State() == "Recording" || (s.Recorder.State() == "Paused" && s.Recorder.Values["Recorder.record"] != 0)
		if recording {
			i.Recording.ToVST = false
		}
	}
	return i
}

// EffectiveIntent resolves saved choices against the latest host observation.
func EffectiveIntent(c config.Config, s model.Snapshot) *config.Intent {
	return ResolveIntent(c.VoiceIntent(), s)
}
