//go:build windows

package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestRedirectedCommandNeedsNoConsole(t *testing.T) {
	if os.Getenv("SNOOFER_CONSOLE_HELPER") == "1" {
		if err := Console(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		window, _, _ := kernel.NewProc("GetConsoleWindow").Call()
		if window != 0 {
			fmt.Fprintln(os.Stderr, "allocated a console for redirected output")
			os.Exit(1)
		}
		fmt.Fprint(os.Stdout, "redirected output")
		os.Exit(0)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestRedirectedCommandNeedsNoConsole$")
	cmd.Env = append(os.Environ(), "SNOOFER_CONSOLE_HELPER=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.DETACHED_PROCESS, HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "redirected output" {
		t.Fatalf("redirected command: %v, %s", err, out)
	}
}
