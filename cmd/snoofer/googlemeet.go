//go:build !core && !no_googlemeet

package main

import "sound-snoofer/plugins/googlemeet"

func init() { plugins = append(plugins, googlemeet.Plugin()) }
