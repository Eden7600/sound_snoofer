//go:build !windows || !amd64

package vroverlay

import "image"

func Open(int, int, *image.RGBA) (*Session, error) { return nil, ErrUnsupported }
