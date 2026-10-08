package control

import "sound-snoofer/internal/config"

// processorObserver is implemented by clients that report whether the voice
// processor is running.
type processorObserver interface {
	SetProcessorProcess(name string)
}

// ConfigureClient applies configuration-derived observation settings to a
// client. Clients without such settings are left unchanged.
func ConfigureClient(client any, cfg config.Config) {
	if observer, ok := client.(processorObserver); ok {
		observer.SetProcessorProcess(cfg.ProcessorProcess())
	}
}
