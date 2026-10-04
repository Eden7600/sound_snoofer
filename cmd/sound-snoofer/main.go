package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"sound-snoofer/internal/cli"
	"sound-snoofer/internal/desktop"
)

func main() { os.Exit(run()) }

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	args := os.Args[1:]
	if len(args) == 2 && args[0] == "__controls" {
		if err := desktop.RunControls(ctx, args[1]); err != nil {
			desktop.ShowError(err)
			return 1
		}
		return 0
	}
	if desktop.IsTrayLaunch(args) {
		if err := desktop.RunTray(ctx, args); err != nil {
			desktop.ShowError(err)
			return 1
		}
		return 0
	}
	if err := desktop.Console(); err != nil {
		desktop.ShowError(err)
		return 1
	}
	return cli.Run(ctx, args, os.Stdout, os.Stderr, cli.DefaultDeps())
}
