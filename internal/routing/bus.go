package routing

import (
	"strconv"
	"strings"
)

// HardwareBus parses an A-prefixed numeric target into its 1-based bus number.
// Callers requiring canonical A1-A5 names must also check the target length.
func HardwareBus(target string) (int, bool) {
	number, ok := strings.CutPrefix(target, "A")
	if !ok {
		return 0, false
	}
	bus, err := strconv.Atoi(number)
	if err != nil || bus < 1 || bus > 5 {
		return 0, false
	}
	return bus, true
}
