// Package camera wraps snoofer-camera.dll: sized property requests to a UVC
// camera's extension units through the camera's own driver. It knows nothing
// about any camera model; callers supply property sets and selectors.
package camera

import "errors"

// GUID is a Windows GUID in its in-memory layout.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	// ErrAbsent means no device matched.
	ErrAbsent = errors.New("camera not present")
	// ErrAmbiguous means several devices matched; none is chosen.
	ErrAmbiguous = errors.New("several matching cameras")
	// ErrNoControl means no extension unit implements the property set.
	ErrNoControl = errors.New("camera control not found")
)
