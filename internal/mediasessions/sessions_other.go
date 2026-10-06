//go:build !windows

package mediasessions

import "errors"

var errUnsupported = errors.New("media sessions unsupported")

// Client is unavailable outside Windows.
type Client struct{}

// Open always fails outside Windows.
func Open(string) (*Client, error) { return nil, errUnsupported }

func (*Client) Snapshot() ([]Session, error)    { return nil, errUnsupported }
func (*Client) Art(string) ([]byte, error)      { return nil, errUnsupported }
func (*Client) Command(string, Op, int64) error { return errUnsupported }
func (*Client) Close() error                    { return nil }
