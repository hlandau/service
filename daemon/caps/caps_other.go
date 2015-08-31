// +build !linux

package caps

const PlatformSupportsCaps = false

func EnsureNoCaps() error {
	return nil
}

func DropCaps() error {
	return nil
}
