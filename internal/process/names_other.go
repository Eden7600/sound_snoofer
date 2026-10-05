//go:build !windows

package process

import "fmt"

// Names is unsupported outside Windows.
func Names() ([]string, error) { return nil, fmt.Errorf("process observation requires Windows") }
