package model

import (
	"reflect"
	"testing"
)

func TestInventoryDevices(t *testing.T) {
	physical := []Device{{Name: "Universal Audio Volt", Driver: "asio"}, {Name: "Microphone (Insta360 Link 2)", ID: `USB\Class_01`}, {Name: "Speakers (SteelSeries Arena 9)"}, {Name: "Unknown ASIO", Driver: "asio"}}
	devices := append([]Device{}, physical...)
	devices = append(devices, Device{Name: "Renamed cable", ID: "VBAudioVACWDM"}, Device{Name: "CABLE Input (VB-Audio Virtual C"}, Device{Name: "Voicemeeter Potato Insert Virtual ASIO", Driver: "asio"}, Device{Name: "VOICEMEETER VAIO3 Virtual ASIO"}, Device{Name: "SteelSeries Sonar - Gaming"})
	before := append([]Device{}, devices...)
	got := InventoryDevices(devices)
	if !reflect.DeepEqual(got, physical) {
		t.Fatalf("got %v, want %v", got, physical)
	}
	got[0].Name = "changed"
	if !reflect.DeepEqual(devices, before) {
		t.Fatal("filter aliases or mutates internal inventory")
	}
	if got := InventoryDevices(devices[len(physical):]); got == nil || len(got) != 0 {
		t.Fatal("virtual-only inventory should be an empty array")
	}
}
