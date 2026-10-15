//go:build windows

package mediasessions

import (
	"errors"
	"testing"
)

func TestNativeTimeoutHasTypedOutcome(t *testing.T) {
	if !errors.Is(result(0x800705B4), ErrTimeout) {
		t.Fatal("native timeout lost its typed identity")
	}
	if result(0) != nil || result(1) != nil {
		t.Fatal("successful HRESULT rejected")
	}
	if err := result(0x80004005); err == nil || errors.Is(err, ErrTimeout) {
		t.Fatal("other native error misclassified", err)
	}
}
