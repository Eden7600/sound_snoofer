//go:build windows && amd64

package voicemeeter

import "sound-snoofer/internal/process"

func processNames() ([]string, error) { return process.Names() }
