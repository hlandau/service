// +build !windows

// Package passwd facilitates the resolution of user and group names and
// membership.
package passwd

import "strconv"

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
	return getGIDForUID(uid)
}

// Given a group ID, returns an array of the supplementary group IDs that group
// implies.
func GetExtraGIDs(gid int) (gids []int, err error) {
	return getExtraGIDs(gid)
}
