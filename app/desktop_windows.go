//go:build windows

package app

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wwindows "github.com/wailsapp/wails/v2/pkg/options/windows"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed web/*
var desktopAssets embed.FS

// Desktop exposes only immutable state and the existing semantic action pipe.
type Desktop struct {
	mu      sync.Mutex
	state   ViewState
	actions chan UIAction
	ctx     context.Context
}

func (d *Desktop) State() ViewState {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.state // Replaced wholesale; snapshots are never mutated.
}
func (d *Desktop) Send(action UIAction) error {
	select {
	case <-d.ctx.Done():
		return errors.New("controls disconnected")
	default:
	}
	select {
	case d.actions <- action:
		return nil
	default:
		return errors.New("controls busy; try again")
	}
}

func RunDesktop(ctx context.Context) error {
	assets, err := fs.Sub(desktopAssets, "web")
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	d := &Desktop{ctx: runCtx, actions: make(chan UIAction, 16)}
	var workers sync.WaitGroup
	err = wails.Run(&options.App{
		Title: "Snoofer", Width: 1280, Height: 820, MinWidth: 800, MinHeight: 600,
		BackgroundColour:   options.NewRGB(14, 20, 27),
		AssetServer:        &assetserver.Options{Assets: assets},
		Logger:             logger.NewFileLogger(filepath.Join(filepath.Dir(exe), "snoofer-gui.log")),
		LogLevelProduction: logger.ERROR,
		Windows:            &wwindows.Options{Theme: wwindows.Dark, IsZoomControlEnabled: true, WebviewUserDataPath: filepath.Join(filepath.Dir(exe), "webview-cache")},
		Bind:               []interface{}{d},
		OnDomReady: func(uiCtx context.Context) {
			workers.Add(3)
			go func() {
				defer workers.Done()
				defer cancel()
				decoder := json.NewDecoder(os.Stdin)
				for {
					var f frame
					if err := decoder.Decode(&f); err != nil {
						return
					}
					if f.State != nil {
						d.mu.Lock()
						d.state = *f.State
						d.mu.Unlock()
					}
					if f.Focus {
						wruntime.WindowUnminimise(uiCtx)
						wruntime.WindowShow(uiCtx)
					}
				}
			}()
			go func() {
				defer workers.Done()
				encoder := json.NewEncoder(os.Stdout)
				for {
					select {
					case <-runCtx.Done():
						return
					case a := <-d.actions:
						if err := encoder.Encode(a); err != nil {
							cancel()
							return
						}
					}
				}
			}()
			go func() { defer workers.Done(); <-runCtx.Done(); wruntime.Quit(uiCtx) }()
		},
		OnShutdown: func(context.Context) { cancel(); os.Stdin.Close(); os.Stdout.Close() },
	})
	cancel()
	os.Stdin.Close()
	os.Stdout.Close()
	workers.Wait()
	return err
}
