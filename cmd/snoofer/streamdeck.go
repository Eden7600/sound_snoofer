//go:build !core && !no_streamdeck

package main

import "sound-snoofer/plugins/streamdeck"

func init() { plugins = append(plugins, streamdeck.Plugin()) }
