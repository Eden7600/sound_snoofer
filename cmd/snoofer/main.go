package main

import (
	"context"
	"fmt"
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
		if clipboardWorkerInvocation(os.Args[1:]) {
			fmt.Fprintln(os.Stderr, err)
		} else {
			app.ShowError(err)
		}
		os.Exit(1)
	}
}

// Worker startup failures must not open a dialog in an active desktop/VR session.
func clipboardWorkerInvocation(args []string) bool {
	return len(args) == 4 && args[0] == "--config" && args[1] != "" && args[2] == "vrclipboard" && args[3] == "worker"
}
