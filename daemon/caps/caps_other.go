// +build !linux

package caps

const platformSupportsCaps = false

func haveAny() bool {
	return false
}

func drop() error {
	return nil
}
