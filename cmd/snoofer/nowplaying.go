//go:build !core && !no_nowplaying

package main

import "sound-snoofer/plugins/nowplaying"

func init() { plugins = append(plugins, nowplaying.Plugin()) }
