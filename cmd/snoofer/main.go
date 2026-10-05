package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"sound-snoofer/app"
	"sound-snoofer/snoofer"
)

var plugins []snoofer.Plugin

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := app.Run(ctx, os.Args[1:], plugins...); err != nil {
		app.ShowError(err)
		os.Exit(1)
	}
}
