// +build linux

package dupfd

import "syscall"

// Always use dup3 on Linux because dup2 is not available on arm64.
func dup2(sourceFD, targetFD int) error {
	return syscall.Dup3(sourceFD, targetFD, 0)
}
