// +build !windows

// Package passwd facilitates the resolution of user and group names and
// membership.
package passwd

import (
	"fmt"
	"strconv"
	"unsafe"
)

/*
#include <sys/types.h>
#include <stdlib.h>
int de_username_to_uid(const char *username, uid_t *uid);
int de_groupname_to_gid(const char *groupname, gid_t *gid);
int de_get_extra_gids(gid_t gid, void *ptr);
int de_gid_for_uid(uid_t uid, gid_t *gid);
*/
import "C"

// Parse a UID string. The string should either be a username or a decimal user
// ID. Returns the user ID or an error.
func ParseUID(uid string) (int, error) {
	n, err := strconv.ParseUint(uid, 10, 31)
	if err != nil {
		return parseUserName(uid)
	}
	return int(n), nil
}

// Parse a GID string. The string should either be a group name or a decimal group
// ID. Returns the group ID or an error.
func ParseGID(gid string) (int, error) {
	n, err := strconv.ParseUint(gid, 10, 31)
	if err != nil {
		return parseGroupName(gid)
	}
	return int(n), nil
}

// Given a UID string (a username or decimal user ID string), find the primary
// GID for the given UID and return it.
func GetGIDForUID(uid string) (int, error) {
	var x C.gid_t
	n, err := ParseUID(uid)
	if err != nil {
		return 0, err
	}
	uidn := C.uid_t(n)
	if C.de_gid_for_uid(uidn, &x) < 0 {
		return 0, fmt.Errorf("cannot get GID for UID: %d", n)
	}
	return int(x), nil
}

//export de_gid_cb
func de_gid_cb(p unsafe.Pointer, gid C.gid_t) {
	f := *(*func(C.gid_t))(p)
	f(gid)
}

// Given a group ID, returns an array of the supplementary group IDs that group
// implies.
func GetExtraGIDs(gid int) (gids []int, err error) {
	gidn := C.gid_t(gid)

	f := func(gid C.gid_t) {
		gids = append(gids, int(gid))
	}

	if C.de_get_extra_gids(gidn, unsafe.Pointer(&f)) < 0 {
		return nil, fmt.Errorf("cannot retrieve additional groups list for GID %d", gid)
	}

	return
}

func parseUserName(username string) (int, error) {
	var x C.uid_t
	cusername := C.CString(username)
	defer C.free(unsafe.Pointer(cusername))

	if C.de_username_to_uid(cusername, &x) < 0 {
		return 0, fmt.Errorf("cannot convert username to uid: %s", username)
	}
	return int(x), nil
}

func parseGroupName(groupname string) (int, error) {
	var x C.gid_t
	cgroupname := C.CString(groupname)
	defer C.free(unsafe.Pointer(cgroupname))

	if C.de_groupname_to_gid(cgroupname, &x) < 0 {
		return 0, fmt.Errorf("cannot convert group name to gid: %s", groupname)
	}
	return int(x), nil
}
