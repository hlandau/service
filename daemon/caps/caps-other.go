// +build !linux linux,!cgo

package caps

const platformSupportsCaps = false

func haveAny() bool {
	return false
}

func drop() error {
	return nil
}
