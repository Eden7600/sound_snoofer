package voicemeeter

import "fmt"

// InsertHook names native insert stages that the monitor's audio callback
// calls on Voicemeeter's audio thread: Input for the input insert, Output for
// the output insert, both with Context. The functions and context must stay
// valid until SetCallback has removed the hook successfully.
type InsertHook struct {
	Input, Output, Context uintptr
}

// SetCallback registers the audio callback when monitoring or an insert hook
// is wanted, and re-registers it when the hook changes. It is called only by
// the live worker holding writer ownership.
func (c *Client) SetCallback(monitor bool, insert *InsertHook) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("Voicemeeter client is closed")
	}
	if callback, ok := c.api.(interface {
		SetCallback(bool, *InsertHook) error
	}); ok {
		return callback.SetCallback(monitor, insert)
	}
	if monitor || insert != nil {
		return fmt.Errorf("audio callback unsupported by this backend")
	}
	return nil
}
