// +build !linux,!windows

package setuid

import "syscall"

func setuid(uid int) error {
	return syscall.Setuid(uid)
}

func setgid(gid int) error {
	return syscall.Setgid(gid)
}

func setgroups(gids []int) error {
	return syscall.Setgroups(gids)
}

func setresgid(rgid, egid, sgid int) error {
	return syscall.Setresgid(rgid, egid, sgid)
}

func setresuid(ruid, euid, suid int) error {
	return syscall.Setresuid(ruid, euid, suid)
}
