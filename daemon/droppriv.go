// +build !windows

package daemon

import (
	"errors"
	"fmt"
	"gopkg.in/hlandau/service.v1/daemon/caps"
	"gopkg.in/hlandau/service.v1/daemon/setuid"
	"gopkg.in/hlandau/service.v1/passwd"
	"net"
	"syscall"
)

// Drops privileges to the specified UID and GID.
// This function does nothing and returns no error if all E?[UG]IDs are nonzero.
//
// If chrootDir is not empty, the process is chrooted into it. The directory
// must exist. The function tests that privilege dropping has been successful
// by attempting to setuid(0), which must fail.
//
// The current directory is set to / inside the chroot.
//
// The function ensures that /etc/hosts and /etc/resolv.conf are loaded before
// chrooting, so name service should continue to be available.
func DropPrivileges(UID, GID int, chrootDir string) (chrootErr error, err error) {
	// chroot and set UID and GIDs
	chrootErr, err = dropPrivileges(UID, GID, chrootDir)
	if err != nil {
		err = fmt.Errorf("dropPrivileges failed: %v", err)
		return
	}

	err = syscall.Chdir("/")
	if err != nil {
		return
	}

	err = ensureNoPrivs()
	if err != nil {
		err = fmt.Errorf("ensure no privs failed: %v", err)
		return
	}

	return
}

func dropPrivileges(UID, GID int, chrootDir string) (chrootErr error, err error) {
	if (UID <= 0) != (GID <= 0) {
		return nil, errors.New("either both or neither UID and GID must be set to positive (i.e. valid, non-root) values")
	}

	var gids []int
	if UID > 0 {
		gids, err = passwd.GetExtraGIDs(GID)
		if err != nil {
			return nil, err
		}

		gids = append(gids, GID)
	}

	chrootErr = tryChroot(chrootDir)

	if UID > 0 {
		err = tryDropPrivileges(UID, GID, gids)
		if err != nil {
			return
		}
	}

	if caps.PlatformSupportsCaps {
		// If we still have any caps (maybe because we didn't setuid), try and drop them.
		err = caps.Drop()
		if err != nil {
			err = fmt.Errorf("cannot drop caps: %v", err)
			return
		}
	}

	return
}

func tryDropPrivileges(UID, GID int, gids []int) error {
	if UID <= 0 || GID <= 0 {
		return errors.New("invalid UID/GID specified so cannot setuid/setgid")
	}

	err := setuid.Setgroups(gids)
	if err != nil {
		return err
	}

	err = setuid.Setresgid(GID, GID, GID)
	if err != nil {
		return err
	}

	err = setuid.Setresuid(UID, UID, UID)
	if err != nil {
		return err
	}

	return nil
}

func tryChroot(path string) error {
	if path == "/" {
		path = ""
	}

	if path == "" {
		return nil
	}

	ensureResolverConfigIsLoaded()

	err := syscall.Chroot(path)
	if err != nil {
		return err
	}

	return nil
}

func ensureResolverConfigIsLoaded() {
	c, err := net.Dial("udp", "un_localhost:1")
	if err == nil {
		c.Close()
	}
}

func ensureNoPrivs() error {
	if IsRoot() {
		return errors.New("still have non-zero UID or GID or capabilities")
	}

	err := setuid.Setuid(0)
	if err == nil {
		return errors.New("Can't drop privileges - setuid(0) still succeeded")
	}

	err = setuid.Setgid(0)
	if err == nil {
		return errors.New("Can't drop privileges - setgid(0) still succeeded")
	}

	return nil
}

// Returns true if either or both of the following are true:
//
// Any of the UID, EUID, GID or EGID are zero.
//
// On supported platforms which support capabilities (currently Linux), any
// capabilities are present.
func IsRoot() bool {
	return caps.HaveAny() || isRoot()
}

func isRoot() bool {
	return syscall.Getuid() == 0 || syscall.Geteuid() == 0 ||
		syscall.Getgid() == 0 || syscall.Getegid() == 0
}

// This is set to a path which should be empty on the target platform.
//
// On Linux, the FHS provides that "/var/empty" should always be empty.
var EmptyChrootPath = "/var/empty"
