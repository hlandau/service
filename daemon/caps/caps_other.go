// +build !linux

package caps

const platformSupportsCaps = false

func haveAny() bool {
	return nil
}

func dropCaps() error {
	return nil
}
