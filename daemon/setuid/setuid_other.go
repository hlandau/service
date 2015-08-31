// +build !linux

package setuid

import "syscall"

// Linux has a faulty, non-compliant implementation of setuid(2) which
// only changes the UID of the current thread, not the whole process.
// Amazingly, even the manual page lies and claims that it affects the
// process.
//
// glibc's setuid syscall wrapper dispatches setuid calls to all
// threads.
//
// The same also applies to setgid, setresuid, setresgid, etc.
// Though oddly enough not setgroups.
//
// Therefore setuid, setgid, setresuid and setresgid are dispatched
// through cgo.

func Setuid(uid int) error {
	return syscall.Setuid(uid)
}

func Setgid(gid int) error {
	return syscall.Setgid(gid)
}

func Setgroups(gids []int) error {
	return syscall.Setgroups(gids)
}

func Setresgid(rgid, egid, sgid int) error {
	return syscall.Setresgid(rgid, egid, sgid)
}

func Setresuid(ruid, euid, suid int) error {
	return syscall.Setresuid(ruid, euid, suid)
}
