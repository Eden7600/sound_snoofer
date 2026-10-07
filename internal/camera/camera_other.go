//go:build !windows

package camera

import "errors"

var errUnsupported = errors.New("camera controls unsupported")

// Library is unavailable outside Windows.
type Library struct{}

// Device is unavailable outside Windows.
type Device struct{}

// Load always fails outside Windows.
func Load(string) (*Library, error) { return nil, errUnsupported }

func (*Library) Release() error                  { return nil }
func (*Library) Open(string) (*Device, error)    { return nil, errUnsupported }
func (*Device) Length(GUID, uint32) (int, error) { return 0, errUnsupported }
func (*Device) Get(GUID, uint32) ([]byte, error) { return nil, errUnsupported }
func (*Device) Set(GUID, uint32, []byte) error   { return errUnsupported }
func (*Device) Close() error                     { return nil }
