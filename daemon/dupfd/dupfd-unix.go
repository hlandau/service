// +build !windows,!solaris,!plan9

package dupfd

import "syscall"

func dup2(sourceFD, targetFD int) error {
	return syscall.Dup2(sourceFD, targetFD)
}
