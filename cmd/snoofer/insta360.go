//go:build !core && !no_insta360

package main

import "sound-snoofer/plugins/insta360"

func init() { plugins = append(plugins, insta360.Plugin()) }
