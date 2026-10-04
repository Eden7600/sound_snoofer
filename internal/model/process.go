package model

// ProcessStatus distinguishes an absent process from a failed observation.
type ProcessStatus struct {
	Known   bool   `json:"known"`
	Running bool   `json:"running"`
	Error   string `json:"error,omitempty"`
}

func (s Snapshot) ElementRunning() bool {
	return s.Element != nil && s.Element.Known && s.Element.Running
}
func (s Snapshot) ElementStatus() string {
	if s.Element == nil || !s.Element.Known {
		return "Element status unknown"
	}
	if s.Element.Running {
		return "Element running"
	}
	return "Element closed"
}
