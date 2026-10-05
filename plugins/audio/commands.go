package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sound-snoofer/internal/cli"
	"sound-snoofer/internal/config"
	"sound-snoofer/snoofer"
)

func command(ctx context.Context, s snoofer.Services, raw json.RawMessage, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("audio commands: devices, plan, apply, watch; use the Snoofer controls window for interactive settings")
	}
	if args[0] == "tui" {
		return fmt.Errorf("open the unified Snoofer controls window from the tray")
	}
	var settings Settings
	if err := snoofer.DecodeSettings(raw, &settings); err != nil {
		return err
	}
	if !filepath.IsAbs(settings.StatePath) {
		settings.StatePath = filepath.Join(filepath.Dir(s.Path), settings.StatePath)
	}
	cfg, err := config.Decode(settings.Config)
	if err != nil {
		return err
	}
	deps := cli.DefaultDeps()
	deps.Load = func(string) (config.Config, error) { return config.LoadChoices(settings.StatePath, cfg), nil }
	for _, arg := range args[1:] {
		if arg == "--config" || arg == "-config" || strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "-config=") {
			return fmt.Errorf("pass --config before audio to select the Snoofer envelope")
		}
	}
	if args[0] != "devices" && args[0] != "help" && args[0] != "--help" {
		args = append(args, "--config", settings.StatePath)
	}
	if !s.Live {
		if args[0] == "apply" {
			return fmt.Errorf("preview mode cannot apply routing; use plan")
		}
		if args[0] == "watch" {
			args = append(args, "--dry-run")
		}
	}
	if settings.DLL != "" {
		args = append(args, "--dll", settings.DLL)
	}
	if code := cli.Run(ctx, args, os.Stdout, os.Stderr, deps); code != 0 {
		return fmt.Errorf("audio command exited with status %d", code)
	}
	return nil
}
