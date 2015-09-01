// Package caps provides functions for controlling capabilities.
package caps

// This constant will be true iff the target platform supports capabilities.
const PlatformSupportsCaps = platformSupportsCaps

// Returns true iff there are no capabilities available to the program.
func HaveAny() bool {
	return haveAny()
}

// Attempt to drop all capabilities.
func Drop() error {
	return drop()
}
