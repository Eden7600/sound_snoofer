//go:build !core && !no_huesync

package main

import "sound-snoofer/plugins/huesync"

func init() { plugins = append(plugins, huesync.Plugin()) }
