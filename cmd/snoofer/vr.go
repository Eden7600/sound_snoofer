//go:build !core && !no_audio && !no_vr

package main

import "sound-snoofer/plugins/vr"

func init() { plugins = append(plugins, vr.Plugin()) }
