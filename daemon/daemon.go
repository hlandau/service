// +build !windows

// Functions to assist with the writing of UNIX-style daemons in go.
package daemon

import "syscall"
import "net"
import "os"
import "errors"
import "gopkg.in/hlandau/service.v1/passwd"
import "gopkg.in/hlandau/service.v1/exepath"
import "gopkg.in/hlandau/service.v1/daemon/setuid"
import "gopkg.in/hlandau/service.v1/daemon/caps"
import "fmt"

// Initialises a daemon with recommended values. Called by Daemonize.
//
// Currently, this only calls umask(0) and chdir("/").
func Init() error {
	syscall.Umask(0)

	err := syscall.Chdir("/")
	if err != nil {
		return err
	}

	// setrlimit RLIMIT_CORE
	return nil
}

const forkedArg = "$*_FORKED_*$"

// Psuedo-forks by re-executing the current binary with a special command line
// argument telling it not to re-execute itself again. Returns true in the
// parent process and false in the child.
func Fork() (isParent bool, err error) {
	if os.Args[len(os.Args)-1] == forkedArg {
		os.Args = os.Args[0 : len(os.Args)-1]
		return false, nil
	}

	newArgs := make([]string, 0, len(os.Args))
	newArgs = append(newArgs, exepath.AbsExePath)
	newArgs = append(newArgs, os.Args[1:]...)
	newArgs = append(newArgs, forkedArg)

	proc, err := os.StartProcess(exepath.AbsExePath, newArgs, &os.ProcAttr{})
	if err != nil {
		return true, err
	}

	proc.Release()
	return true, nil
}

// Daemonizes but doesn't fork.
//
// The stdin, stdout and stderr fds are remapped to /dev/null.
// setsid is called.
//
// The process changes its current directory to /.
//
// If you intend to call DropPrivileges, call it after calling this function,
// as /dev/null will no longer be available after privileges are dropped.
func Daemonize() error {
	null_f, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer null_f.Close()

	stdin_fd := int(os.Stdin.Fd())
	stdout_fd := int(os.Stdout.Fd())
	stderr_fd := int(os.Stderr.Fd())

	// ... reopen fds 0, 1, 2 as /dev/null ...
	// Since dup2 closes fds which are already open we needn't close the above fds.
	// This lets us avoid race conditions.
	null_fd := int(null_f.Fd())
	err = syscall.Dup2(null_fd, stdin_fd)
	if err != nil {
		return err
	}

	err = syscall.Dup2(null_fd, stdout_fd)
	if err != nil {
		return err
	}

	err = syscall.Dup2(null_fd, stderr_fd)
	if err != nil {
		return err
	}

	// This may fail if we're not root
	syscall.Setsid()

	// Daemonize implies Init.
	return Init()
}

// Returns true if either or both of the following are true:
//
// Any of the UID, EUID, GID or EGID are zero.
//
// On supported platforms which support capabilities (currently Linux), any
// capabilities are present.
func IsRoot() bool {
	return caps.EnsureNoCaps() != nil || isRoot()
}

func isRoot() bool {
	return syscall.Getuid() == 0 || syscall.Geteuid() == 0 ||
		syscall.Getgid() == 0 || syscall.Getegid() == 0
}

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
	err = setRlimits()
	if err != nil {
		err = fmt.Errorf("failed to set rlimits: %v", err)
		return
	}

	err = platformPreDropPrivileges()
	if err != nil {
		err = fmt.Errorf("platformPreDropPrivileges failed: %v", err)
		return
	}

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

	err = platformPostDropPrivileges()
	if err != nil {
		err = fmt.Errorf("platformPostDropPrivileges failed: %v", err)
		return
	}

	err = nil
	return
}

func setRlimits() error {
	// TODO
	return nil
}

func dropPrivileges(UID, GID int, chrootDir string) (chrootErr error, err error) {
	if (UID == -1) != (GID == -1) {
		return nil, errors.New("either both or neither UID and GID must be -1")
	}

	if isRoot() {
		if UID <= 0 || GID <= 0 {
			return nil, errors.New("must specify UID/GID when running as root")
		}
	}

	var gids []int
	if UID != -1 {
		gids, err = passwd.GetExtraGIDs(GID)
		if err != nil {
			return nil, err
		}
	}

	chrootErr = tryChroot(chrootDir)

	gids = append(gids, GID)

	err = tryDropPrivileges(UID, GID, gids)
	if err == errZeroUID {
		return
	} else if err != nil {
		if caps.PlatformSupportsCaps {
			// We can't setuid, so maybe we only have a few caps.
			// Drop them.
			err = caps.DropCaps()
			if err != nil {
				err = fmt.Errorf("cannot drop caps: %v", err)
			}
			return
		} else {
			return
		}
	}

	err = nil
	return
}

var errZeroUID = errors.New("Can't drop privileges to UID/GID0 - did you set the UID/GID properly?")

func tryDropPrivileges(UID, GID int, gids []int) error {
	if UID == -1 {
		return errors.New("invalid UID specified so cannot setuid")
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

	if UID == 0 || GID == 0 {
		return errZeroUID
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
	if err != nil {

	} else {
		c.Close()
	}
}

func ensureNoPrivs() error {
	if isRoot() {
		return errors.New("still have non-zero UID or GID")
	}

	err := setuid.Setuid(0)
	if err == nil {
		return errors.New("Can't drop privileges - setuid(0) still succeeded")
	}

	err = setuid.Setgid(0)
	if err == nil {
		return errors.New("Can't drop privileges - setgid(0) still succeeded")
	}

	return caps.EnsureNoCaps()
}

var EmptyChrootPath = "/var/empty"
