package setuid

import "syscall"

/*
#define _GNU_SOURCE
#include <unistd.h>
#include <errno.h>

// These differ to the libc prototypes because they return
// 0 on success but errno on failure rather than -1.

static int
csetuid(uid_t uid) {
  int ec = setuid(uid);
  return (ec < 0) ? errno : 0;
}

static int
csetgid(gid_t gid) {
  int ec = setgid(gid);
  return (ec < 0) ? errno : 0;
}

static int
csetresuid(uid_t ruid, uid_t euid, uid_t suid) {
  int ec = setresuid(ruid, euid, suid);
  return (ec < 0) ? errno : 0;
}

static int
csetresgid(gid_t rgid, gid_t egid, gid_t sgid) {
  int ec = setresgid(rgid, egid, sgid);
  return (ec < 0) ? errno : 0;
}

*/
import "C"

// Linux has a faulty, non-compliant implementation of setuid(2) which only
// changes the UID of the current thread, not the whole process. Amazingly,
// even the manual page lies and claims that it affects the process.
//
// glibc's setuid syscall wrapper dispatches setuid calls to all threads. Ergo,
// the manual page for setuid(3) but not setuid(2) is accurate.
//
// The same also applies to setgid, setresuid, setresgid, etc. Though oddly
// enough not setgroups.
//
// Therefore setuid, setgid, setresuid and setresgid are dispatched through
// cgo.

func Setuid(uid int) error {
	eno := C.csetuid(C.uid_t(uid))
	if eno != 0 {
		return syscall.Errno(eno)
	}
	return nil
}

func Setgid(gid int) error {
	eno := C.csetgid(C.gid_t(gid))
	if eno != 0 {
		return syscall.Errno(eno)
	}
	return nil
}

func Setgroups(gids []int) error {
	return syscall.Setgroups(gids)
}

func Setresgid(rgid, egid, sgid int) error {
	eno := C.csetresgid(C.gid_t(rgid), C.gid_t(egid), C.gid_t(sgid))
	if eno != 0 {
		return syscall.Errno(eno)
	}
	return nil
}

func Setresuid(ruid, euid, suid int) error {
	eno := C.csetresuid(C.uid_t(ruid), C.uid_t(euid), C.uid_t(suid))
	if eno != 0 {
		return syscall.Errno(eno)
	}
	return nil
}
