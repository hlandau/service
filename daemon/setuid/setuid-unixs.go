// +build !linux,!windows,!darwin,!freebsd,!openbsd,!netbsd,!solaris,!plan9,!dragonfly

package setuid

import "syscall"

func setuid(uid int) error {
	return syscall.Setuid(uid)
}

func setgid(gid int) error {
	return syscall.Setgid(gid)
}

func setresgid(rgid, egid, sgid int) error {
	return syscall.Setresgid(rgid, egid, sgid)
}

func setresuid(ruid, euid, suid int) error {
	return syscall.Setresuid(ruid, euid, suid)
}
