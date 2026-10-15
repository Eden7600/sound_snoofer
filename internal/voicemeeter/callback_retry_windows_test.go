//go:build windows && amd64

package voicemeeter

import (
	"testing"
	"time"
)

func TestFailedCallbackRegistrationRetries(t *testing.T) {
	now := time.Now()
	// A failed registration leaves the monitor enabled but inactive.
	m := callbackMonitor{enabled: true, retryAt: now.Add(callbackRetry)}
	if !m.settled(true, nil, now) {
		t.Fatal("retried before the retry delay")
	}
	if m.settled(true, nil, now.Add(callbackRetry)) {
		t.Fatal("failed registration never retried")
	}
	m.active = true
	if !m.settled(true, nil, now.Add(time.Hour)) {
		t.Fatal("active registration attempted again")
	}
	if m.settled(false, nil, now) || m.settled(true, &InsertHook{Input: 1}, now) {
		t.Fatal("changed request treated as settled")
	}
}
