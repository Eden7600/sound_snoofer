//go:build !core && !no_hue

package main

import "sound-snoofer/plugins/hue"

func init() { plugins = append(plugins, hue.Plugin()) }
