//go:build !windows

package insta360

import "errors"

type nativeOpener struct{}

func (*nativeOpener) open() (device, error) { return nil, errors.New("camera controls need Windows") }
func (*nativeOpener) release() error        { return nil }
