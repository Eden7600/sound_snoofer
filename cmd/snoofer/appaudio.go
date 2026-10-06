//go:build !core && !no_appaudio

package main

import "sound-snoofer/plugins/appaudio"

func init() { plugins = append(plugins, appaudio.Plugin()) }
