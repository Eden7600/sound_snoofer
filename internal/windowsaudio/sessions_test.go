package windowsaudio

import "testing"

func TestProgramName(t *testing.T) {
	for _, c := range []struct{ path, description, display, want string }{
		{SystemSounds, "", `@%SystemRoot%\System32\AudioSrv.Dll,-202`, "System sounds"},
		{`C:\Apps\Discord\Discord.exe`, "Discord", "", "Discord"},
		{`C:\Games\game.exe`, "", "Game Audio", "Game Audio"},
		{`C:\Program Files\WindowsApps\Spotify.exe`, "", "ms-resource:AppName", "Spotify"},
		{`C:\Tools\tool.exe`, " ", "@{Package?ms-resource://x}", "tool"},
		{"", "", "", "Unknown app"},
	} {
		if got := programName(c.path, c.description, c.display); got != c.want {
			t.Errorf("%q: %q, want %q", c.path, got, c.want)
		}
	}
}
