//go:build !windows || !amd64

package voicemeeter

import "fmt"

func Open(path string) (*Client, error) { return nil, fmt.Errorf("Voicemeeter requires Windows amd64") }
