package voicemeeter

import "fmt"

// SetMonitoring is called only by the live worker holding writer ownership.
func (c *Client) SetMonitoring(enable bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("Voicemeeter client is closed")
	}
	if monitor, ok := c.api.(interface{ SetMonitoring(bool) error }); ok {
		return monitor.SetMonitoring(enable)
	}
	if enable {
		return fmt.Errorf("callback monitoring unsupported by this backend")
	}
	return nil
}
