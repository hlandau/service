// +build linux

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

func setuid(uid int) error {
	eno := C.csetuid(C.uid_t(uid))
	if eno != 0 {
		return syscall.Errno(eno)
	}
	return nil
}

func setgid(gid int) error {
	eno := C.csetgid(C.gid_t(gid))
	if eno != 0 {
		return syscall.Errno(eno)
	}
	return nil
}

func setgroups(gids []int) error {
	return syscall.Setgroups(gids)
}

func setresgid(rgid, egid, sgid int) error {
	eno := C.csetresgid(C.gid_t(rgid), C.gid_t(egid), C.gid_t(sgid))
	if eno != 0 {
		return syscall.Errno(eno)
	}
	return nil
}

func setresuid(ruid, euid, suid int) error {
	eno := C.csetresuid(C.uid_t(ruid), C.uid_t(euid), C.uid_t(suid))
	if eno != 0 {
		return syscall.Errno(eno)
	}
	return nil
}
