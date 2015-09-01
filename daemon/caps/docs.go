// Package caps provides functions for controlling capabilities.
package caps

// This constant will be true iff the target platform supports capabilities.
const PlatformSupportsCaps = platformSupportsCaps

// Ensure that no capabilities are available to the program. Returns error iff
// this is not the case.
func EnsureNone() error {
	return ensureNone()
}

// Attempt to drop all capabilities.
func Drop() error {
	return drop()
}
