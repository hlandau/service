// +build !cgo,!windows

package passwd

import "fmt"

var errNoCgo = fmt.Errorf("because this binary was built without cgo, UIDs/GIDs must be specified numerically and not as names")

func parseUserName(username string) (int, error) {
	return 0, errNoCgo
}

func parseGroupName(groupname string) (int, error) {
	return 0, errNoCgo
}

func getGIDForUID(uid string) (int, error) {
	n, err := ParseUID(uid)
	if err != nil {
		return 0, err
	}

	// XXX: assume GID is same as UID
	return n, nil
}

func getExtraGIDs(gid int) (gids []int, err error) {
	// XXX: use empty extra GIDs list
	return nil, nil
}
