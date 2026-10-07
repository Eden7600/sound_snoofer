//go:build !windows

package discord

import (
	"errors"
	"io"
)

var errNoDiscord = errors.New("Discord closed")

func dialPipe() (io.ReadWriteCloser, string, error) {
	return nil, "", errors.New("Discord control needs Windows")
}
