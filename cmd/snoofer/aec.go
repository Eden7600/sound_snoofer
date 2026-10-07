//go:build !core && !no_audio && !no_aec

package main

import "sound-snoofer/plugins/aec"

func init() { plugins = append(plugins, aec.Plugin()) }
