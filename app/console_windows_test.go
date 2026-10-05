//go:build windows

package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestControlsConsoleProcess(t *testing.T) {
	if os.Getenv("SNOOFER_CONSOLE_CHILD") == "1" {
		// Standard handles are IPC pipes even when Windows attached a console.
		pipeIn, pipeOut := os.Stdin, os.Stdout
		in, out, err := controlsConsole()
		if err != nil {
			t.Fatal(err)
		}
		defer in.Close()
		defer out.Close()
		var mode uint32
		if err = windows.GetConsoleMode(windows.Handle(in.Fd()), &mode); err != nil {
			t.Fatal(err)
		}
		if err = windows.GetConsoleMode(windows.Handle(out.Fd()), &mode); err != nil {
			t.Fatal(err)
		}
		// Reopening must also tolerate an existing console.
		in2, out2, err := controlsConsole()
		if err != nil {
			t.Fatal(err)
		}
		in2.Close()
		out2.Close()
		data, err := io.ReadAll(pipeIn)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprint(pipeOut, "IPC:"+string(data))
		return
	}
	if os.Getenv("SNOOFER_CONSOLE_PROBE") != "1" {
		t.Skip("requires interactive Windows desktop")
	}
	for _, flags := range []uint32{windows.CREATE_NEW_CONSOLE, windows.DETACHED_PROCESS} {
		t.Run(fmt.Sprint(flags), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			exe, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.CommandContext(ctx, exe, "-test.run=^TestControlsConsoleProcess$")
			cmd.Env = append(os.Environ(), "SNOOFER_CONSOLE_CHILD=1")
			cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flags}
			cmd.Stdin = strings.NewReader("ping")
			output, err := cmd.CombinedOutput()
			if err != nil || !strings.Contains(string(output), "IPC:ping") {
				t.Fatalf("child: %v: %s", err, output)
			}
		})
	}
}
