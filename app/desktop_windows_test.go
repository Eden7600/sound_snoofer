//go:build windows

package app

import (
	"context"
	"testing"
)

func TestDesktopBoundedActions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	d := Desktop{ctx: ctx, actions: make(chan UIAction, 1)}
	if err := d.Send(UIAction{Kind: "retry"}); err != nil {
		t.Fatal(err)
	}
	if err := d.Send(UIAction{Kind: "retry"}); err == nil {
		t.Fatal("unbounded dispatch")
	}
	<-d.actions
	cancel()
	if err := d.Send(UIAction{Kind: "retry"}); err == nil {
		t.Fatal("disconnected dispatch")
	}
}
