//go:build !core && !no_audio

package main

import "sound-snoofer/plugins/audio"

func init() { plugins = append(plugins, audio.Plugin()) }
