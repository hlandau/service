// +build !windows

package passwd

import "strconv"
import "fmt"
import "unsafe"

/*
#include <sys/types.h>
#include <stdlib.h>
int de_username_to_uid(const char *username, uid_t *uid);
int de_groupname_to_gid(const char *groupname, gid_t *gid);
int de_get_extra_gids(gid_t gid, void *ptr);
int de_gid_for_uid(uid_t uid, gid_t *gid);
*/
import "C"

func ParseUID(uid string) (int, error) {
	n, err := strconv.ParseUint(uid, 10, 31)
	if err != nil {
		return parseUserName(uid)
	}
	return int(n), nil
}

func ParseGID(gid string) (int, error) {
	n, err := strconv.ParseUint(gid, 10, 31)
	if err != nil {
		return parseGroupName(gid)
	}
	return int(n), nil
}

func GetGIDForUID(uid string) (int, error) {
	var x C.gid_t
	n, err := ParseUID(uid)
	if err != nil {
		return 0, err
	}
	uid_ := C.uid_t(n)
	if C.de_gid_for_uid(uid_, &x) < 0 {
		return 0, fmt.Errorf("cannot get GID for UID: %d", n)
	}
	return int(x), nil
}

//export de_gid_cb
func de_gid_cb(p unsafe.Pointer, gid C.gid_t) {
	f := *(*func(C.gid_t))(p)
	f(gid)
}

func GetExtraGIDs(gid int) (gids []int, err error) {
	gid_ := C.gid_t(gid)

	f := func(gid C.gid_t) {
		gids = append(gids, int(gid))
	}

	if C.de_get_extra_gids(gid_, unsafe.Pointer(&f)) < 0 {
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
