// +build !linux

package caps

const platformSupportsCaps = false

func ensureNoCaps() error {
	return nil
}

func dropCaps() error {
	return nil
}
