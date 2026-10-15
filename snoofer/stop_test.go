package snoofer

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCancelAndWait(t *testing.T) {
	t.Run("cancel before joining", func(t *testing.T) {
		run, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan struct{})
		go func() {
			<-run.Done()
			close(done)
		}()
		stop, stopCancel := context.WithTimeout(context.Background(), time.Second)
		defer stopCancel()
		if err := CancelAndWait(stop, cancel, done); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("already completed", func(t *testing.T) {
		done := make(chan struct{})
		close(done)
		cancelled := false
		if err := CancelAndWait(context.Background(), func() { cancelled = true }, done); err != nil || !cancelled {
			t.Fatal("completed worker did not cancel and join", err, cancelled)
		}
	})
	t.Run("deadline", func(t *testing.T) {
		stop, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		cancelled := false
		err := CancelAndWait(stop, func() { cancelled = true }, make(chan struct{}))
		if !errors.Is(err, context.DeadlineExceeded) || !cancelled {
			t.Fatal("deadline did not cancel and return", err, cancelled)
		}
	})
}
