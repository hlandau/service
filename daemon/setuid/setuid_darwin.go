// +build darwin

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

// darwin doesn't seem to have saved GIDs, so use setregid
func setresgid(rgid, egid, sgid int) error {
	return syscall.Setregid(rgid, egid)
}

// darwin doesn't seem to have saved UIDs, so use setreuid
func setresuid(ruid, euid, suid int) error {
	return syscall.Setreuid(ruid, euid)
}
