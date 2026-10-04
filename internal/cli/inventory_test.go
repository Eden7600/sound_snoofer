package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"sound-snoofer/internal/model"
)

func TestDevicesOmitsVirtualEndpoints(t *testing.T) {
	for _, asJSON := range []bool{false, true} {
		client, deps, _, _ := setup(t)
		client.s.Devices = append(client.s.Devices, model.Device{Name: "Voicemeeter Virtual ASIO"})
		args := []string{"devices"}
		if asJSON {
			args = append(args, "--json")
		}
		var out, errout bytes.Buffer
		if code := Run(context.Background(), args, &out, &errout, deps); code != 0 {
			t.Fatal(errout.String())
		}
		if strings.Contains(out.String(), "Virtual ASIO") || !strings.Contains(out.String(), "speakers") {
			t.Fatal(out.String())
		}
		if len(client.s.Devices) != 2 || client.writes != 0 {
			t.Fatal("inventory command altered backend")
		}
	}
}
