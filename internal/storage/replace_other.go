//go:build !windows

package storage

import "os"

func replaceState(from, to string) error { return os.Rename(from, to) }
