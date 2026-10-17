package main

import "testing"

func TestClipboardWorkerInvocation(t *testing.T) {
	if !clipboardWorkerInvocation([]string{"--config", "C:/settings.json", "vrclipboard", "worker"}) {
		t.Fatal("worker not recognized")
	}
	for _, args := range [][]string{nil, {"__controls"}, {"--config", "", "vrclipboard", "worker"}, {"--config", "C:/settings.json", "other", "worker"}, {"--config", "C:/settings.json", "vrclipboard"}, {"--config", "C:/settings.json", "vrclipboard", "worker", "extra"}} {
		if clipboardWorkerInvocation(args) {
			t.Fatal("ordinary invocation treated as hidden worker", args)
		}
	}
}
