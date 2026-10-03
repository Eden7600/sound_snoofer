//go:build !windows

package ownership

import "fmt"

func Acquire() (func(), error) { return nil, fmt.Errorf("live routing requires Windows") }
