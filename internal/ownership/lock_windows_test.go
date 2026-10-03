//go:build windows

package ownership

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestWriterOwnershipAndRelease(t *testing.T) {
	name := fmt.Sprintf(`Local\VoiceSnooter.Test.%d.%d`, os.Getpid(), time.Now().UnixNano())
	release, e := acquire(name)
	if e != nil {
		t.Fatal(e)
	}
	if other, e := acquire(name); e == nil {
		other()
		release()
		t.Fatal("second writer accepted")
	}
	release()
	third, e := acquire(name)
	if e != nil {
		t.Fatal(e)
	}
	third()
}
