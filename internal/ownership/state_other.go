//go:build !windows

package ownership

import (
	"fmt"
	"sync"
)

var stateLocks sync.Map

func AcquireState(path string) (func(), error) {
	if _, loaded := stateLocks.LoadOrStore(path, true); loaded {
		return nil, fmt.Errorf("saved choices busy")
	}
	return func() { stateLocks.Delete(path) }, nil
}
