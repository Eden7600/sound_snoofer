//go:build !core && !no_vr && !no_vrclipboard

package main

import "sound-snoofer/plugins/vrclipboard"

func init() { plugins = append(plugins, vrclipboard.Plugin()) }
