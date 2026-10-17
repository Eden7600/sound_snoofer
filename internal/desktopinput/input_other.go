//go:build !windows || !amd64

package desktopinput

import (
	"context"
	"errors"
)

// Capture is unavailable outside the supported Windows amd64 build.
func Capture() (Target, error) {
	return Target{}, errors.New("Desktop shortcuts require Windows amd64")
}

// Send is unavailable outside the supported Windows amd64 build.
func Send(context.Context, Target, Action) error {
	return errors.New("Desktop shortcuts require Windows amd64")
}
