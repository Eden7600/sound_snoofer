package config

import (
	"fmt"
	"strings"
)

// Default executables whose presence Snoofer observes.
const (
	DefaultProcessorProcess = "element.exe"
	DefaultVRProcess        = "vrserver.exe"
)

// ProcessorProcess is the executable whose presence makes Element mode
// available.
func (c Config) ProcessorProcess() string {
	if c.Studio == nil || c.Studio.Voice == nil || c.Studio.Voice.ProcessorProcess == "" {
		return DefaultProcessorProcess
	}
	return c.Studio.Voice.ProcessorProcess
}

// ValidateProcessName accepts a bare executable name; process observation
// compares names case-insensitively and never matches paths.
func ValidateProcessName(name string) error {
	if name == "" || strings.TrimSpace(name) != name {
		return fmt.Errorf("process name %q must be non-empty without surrounding spaces", name)
	}
	if strings.ContainsAny(name, `/\:`) {
		return fmt.Errorf("process name %q must not contain a path", name)
	}
	return nil
}

// ProcessName is the executable whose presence activates the profile.
func (p ProfilePolicy) ProcessName() string {
	if p.Process == "" {
		return DefaultVRProcess
	}
	return p.Process
}
