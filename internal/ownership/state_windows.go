//go:build windows

package ownership

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

func AcquireState(path string) (func(), error) {
	return acquire(fmt.Sprintf(`Local\VoiceSnooter.State.%x`, sha256.Sum256([]byte(strings.ToLower(path)))))
}
