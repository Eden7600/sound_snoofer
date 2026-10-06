package model

// RemoteInfo describes the native Voicemeeter Remote connection for diagnostics.
type RemoteInfo struct {
	DLLPath string // Remote DLL actually loaded; empty for test backends.
	Login   int32  // VBVMR_Login result: 0 Voicemeeter running, 1 launched or not running.
	Version string // Voicemeeter application version; empty when unknown.
}
