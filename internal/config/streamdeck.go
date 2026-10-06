package config

import "fmt"

type StreamDeck struct {
	Profiles []DeckProfile `json:"profiles,omitempty"`
}
type DeckProfile struct {
	Serial string   `json:"serial"`
	Keys   []string `json:"keys"`
}

func (d *StreamDeck) Validate() error {
	seen := map[string]bool{}
	for _, p := range d.Profiles {
		if p.Serial == "" || seen[p.Serial] {
			return fmt.Errorf("Stream Deck profile needs a unique serial")
		}
		seen[p.Serial] = true
		if len(p.Keys) > 36 {
			return fmt.Errorf("Stream Deck + XL has 36 keys")
		}
		for _, key := range p.Keys {
			if !DeckAction(key) {
				return fmt.Errorf("unsupported Stream Deck action %q", key)
			}
		}
	}
	return nil
}
func DeckAction(key string) bool {
	switch key {
	case "", "record-toggle", "record-start", "record-stop", "record-computer", "record-mic", "record-tap", "snippet-play", "snippet-stop", "record-loop", "record-vst", "mic-mute", "speaker-mute", "monitor", "mode", "media-prev", "media-play", "media-next", "media-stop", "open-controls", "engine-restart", "vr-mic", "vr-playback", "defaults":
		return true
	}
	return false
}
