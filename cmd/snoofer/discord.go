//go:build !core && !no_discord

package main

import "sound-snoofer/plugins/discord"

func init() { plugins = append(plugins, discord.Plugin()) }
