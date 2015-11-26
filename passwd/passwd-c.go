// +build cgo,!windows

package passwd

import (
	"fmt"
	"unsafe"
)

/*
#include "pwnam.h"
#include <sys/types.h>
#include <stdlib.h>
*/
import "C"

func getGIDForUID(uid string) (int, error) {
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

func getExtraGIDs(gid int) (gids []int, err error) {
	gidn := C.gid_t(gid)

	f := func(gid C.gid_t) {
		gids = append(gids, int(gid))
	}

	if C.de_get_extra_gids(gidn, unsafe.Pointer(&f)) < 0 {
		return nil, fmt.Errorf("cannot retrieve additional groups list for GID %d", gid)
	}

	return
}
