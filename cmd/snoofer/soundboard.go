//go:build !core && !no_audio && !no_soundboard

package main

import "sound-snoofer/plugins/soundboard"

func init() { plugins = append(plugins, soundboard.Plugin()) }
