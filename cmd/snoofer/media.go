//go:build !core && !no_media

package main

import "sound-snoofer/plugins/media"

func init() { plugins = append(plugins, media.Plugin()) }
