// +build !linux

package bansuid

func banSuid() error {
	return ErrNotSupported
}
