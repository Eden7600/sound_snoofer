package voicemeeter

import (
	"fmt"
	"math"
	"sound-snoofer/internal/model"
)

// GainLevels observes physical playback buses, the active hardware mic and, on
// Potato, the AUX strip (Element's processed return) on the caller's native
// worker thread. Missing entries are unknown, never a fabricated zero reading.
func (c *Client) GainLevels(micStrip int) map[string]float32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.refresh() != nil {
		return nil
	}
	reader, ok := c.api.(interface {
		GetLevel(int, int) (float32, int32)
	})
	if !ok {
		return nil
	}
	edition, code := c.api.Edition()
	if code != 0 || (edition != 2 && edition != 3) {
		return nil
	}
	levels := map[string]float32{}
	read := func(parameter string, kind, first, count int) {
		var peak float32
		for ch := first; ch < first+count; ch++ {
			value, code := reader.GetLevel(kind, ch)
			if code != 0 || value < 0 || math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				return
			}
			peak = max(peak, value)
		}
		levels[parameter] = peak
	}
	buses, _ := model.Limits(int(edition))
	for bus := 0; bus < buses; bus++ {
		read(fmt.Sprintf("Bus[%d].Gain", bus), 3, bus*8, 8)
	}
	physical := 3
	if edition == 3 {
		physical = 5
	}
	if micStrip >= 0 && micStrip < physical {
		read(fmt.Sprintf("Strip[%d].Gain", micStrip), 2, micStrip*2, 2)
	}
	if edition == 3 {
		// AUX is the second virtual strip: 5 physical strips x 2, then 8 per virtual.
		read("Strip[6].Gain", 2, physical*2+8, 2)
	}
	return levels
}

// InputLevels reads the pre-fader input peak (linear) of physical strips on
// the caller's native worker thread, taken before any input insert while one
// is delivering audio. Pre-fader levels ignore strip mute, so a muted
// microphone still reads its signal. Strips with a missing or invalid
// reading are omitted: unknown, never silent.
func (c *Client) InputLevels(strips []int) map[int]float32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.refresh() != nil {
		return nil
	}
	reader, ok := c.api.(interface {
		GetLevel(int, int) (float32, int32)
	})
	if !ok {
		return nil
	}
	edition, code := c.api.Edition()
	if code != 0 || edition < 1 || edition > 3 {
		return nil
	}
	physical := map[int32]int{1: 2, 2: 3, 3: 5}[edition]
	// An input insert such as echo cancellation changes what the strip meters
	// see, so fresh pre-insert peaks take precedence.
	var raw []float32
	if peaks, ok := c.api.(interface {
		InputPeaks(channels int) ([]float32, bool)
	}); ok {
		if values, fresh := peaks.InputPeaks(physical * 2); fresh {
			raw = values
		}
	}
	levels := map[int]float32{}
	for _, strip := range strips {
		if strip < 0 || strip >= physical {
			continue
		}
		var peak float32
		valid := true
		for ch := strip * 2; ch < strip*2+2; ch++ {
			var value float32
			var code int32
			if raw != nil {
				value = raw[ch]
			} else {
				value, code = reader.GetLevel(0, ch)
			}
			if code != 0 || value < 0 || math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				valid = false
				break
			}
			peak = max(peak, value)
		}
		if valid {
			levels[strip] = peak
		}
	}
	return levels
}
