// Package bansuid provides a function to prevent processes from reacquiring privileges.
package bansuid

import "errors"

// BanSuid: On Linux, uses prctl() SECUREBITS and NO_NEW_PRIVS to prevent the process or its descendants
// from ever obtaining privileges by execing a suid/sgid/cap xattr binary. Returns ErrNotSupported
// if platform is not supported. May return other errors.
func BanSuid() error {
	return banSuid()
}

// Returned by BanSuid if it is not supported on the current platform.
var ErrNotSupported = errors.New("bansuid not supported")
