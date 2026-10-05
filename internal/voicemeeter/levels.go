package voicemeeter

import (
	"fmt"
	"math"
)

// GainLevels observes A1/A2 and the active hardware mic on the caller's native
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
	for bus := 0; bus < 2; bus++ {
		read(fmt.Sprintf("Bus[%d].Gain", bus), 3, bus*8, 8)
	}
	physical := 3
	if edition == 3 {
		physical = 5
	}
	if micStrip >= 0 && micStrip < physical {
		read(fmt.Sprintf("Strip[%d].Gain", micStrip), 2, micStrip*2, 2)
	}
	return levels
}
