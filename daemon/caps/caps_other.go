// +build !linux

package caps

const platformSupportsCaps = false

func haveAny() bool {
	return false
}

func dropCaps() error {
	return nil
}
