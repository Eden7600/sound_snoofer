//go:build !windows

package aec

import (
	"errors"

	"sound-snoofer/internal/voicemeeter"
)

var errUnsupported = errors.New("echo cancellation unsupported")

// Engine is unavailable outside Windows.
type Engine struct{}

// Open always fails outside Windows.
func Open(string) (*Engine, error) { return nil, errUnsupported }

func (*Engine) Configure(Config) error       { return errUnsupported }
func (*Engine) Stats() (Stats, error)        { return Stats{}, errUnsupported }
func (*Engine) Reset() error                 { return errUnsupported }
func (*Engine) Hook() voicemeeter.InsertHook { return voicemeeter.InsertHook{} }
func (*Engine) Close() error                 { return nil }
