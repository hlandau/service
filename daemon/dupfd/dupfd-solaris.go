// +build solaris

package dupfd

import "syscall"

func fcntl1(fd uintptr, cmd uintptr, arg uintptr) (val uintptr, err syscall.Errno)

const f_dup2fd = 0x09

func dup2(sourceFD, targetFD int) error {
	_, err := fcntl1(uintptr(sourceFD), f_dup2fd, uintptr(targetFD))
	return err
}
