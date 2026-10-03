//go:build !windows

package config

import "os"

func replaceState(from, to string) error { return os.Rename(from, to) }
