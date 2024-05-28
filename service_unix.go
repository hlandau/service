//go:build !windows
// +build !windows

package service

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/hlandau/service.v3/daemon"
	"gopkg.in/hlandau/service.v3/daemon/bansuid"
	"gopkg.in/hlandau/svcutils.v1/caps"
	"gopkg.in/hlandau/svcutils.v1/passwd"
	"gopkg.in/hlandau/svcutils.v1/pidfile"
	"gopkg.in/hlandau/svcutils.v1/systemd"
)

// This will always point to a path which the platform guarantees is an empty
// directory. You can use it as your default chroot path if your service doesn't
// access the filesystem after it's started.
//
// On Linux, the FHS provides that "/var/empty" is an empty directory, so it
// points to that.
var EmptyChrootPath = daemon.EmptyChrootPath

func usingPlatform(platformName string) bool {
	return platformName == "unix"
}

func systemdUpdateStatus(status string) error {
	return systemd.NotifySend(status)
}

func (info *Info) serviceMain() error {
	if info.Config.Fork {
		isParent, err := daemon.Fork()
		if err != nil {
			return err
		}

		if isParent {
			os.Exit(0)
		}

		info.Config.Daemon = true
	}

	err := daemon.Init()
	if err != nil {
		return err
	}

	err = systemdUpdateStatus("\n")
	if err == nil {
		info.systemd = true
	}

	// default:                   daemon=no,  stderr=yes
	// --daemon:                  daemon=yes, stderr=no
	// systemd/--daemon --stderr: daemon=yes, stderr=yes
	// systemd --daemon:          daemon=yes, stderr=no
	daemonize := info.Config.Daemon
	keepStderr := info.Config.Stderr
	if !daemonize && info.systemd {
		daemonize = true
		keepStderr = true
	}

	if daemonize {
		err := daemon.Daemonize(keepStderr)
		if err != nil {
			return err
		}
	}

	if info.Config.PIDFile != "" {
		info.pidFileName = info.Config.PIDFile

		err = info.openPIDFile()
		if err != nil {
			return err
		}

		defer info.closePIDFile()
	}

	return info.runInteractively()
}

func (info *Info) openPIDFile() error {
	f, err := pidfile.Open(info.pidFileName)
	info.pidFile = f
	return err
}

func (info *Info) closePIDFile() {
	if info.pidFile != nil {
		info.pidFile.Close()
	}
}

func (h *ihandler) DropPrivileges() error {
	if h.dropped {
		return nil
	}

	// Extras
	if !h.info.NoBanSuid {
		// Try and bansuid, but don't process errors. It may not be supported on
		// the current platform, and Linux won't allow SECUREBITS to be set unless
		// one is root (or has the right capability). This is basically a
		// best-effort thing.
		bansuid.BanSuid()
	}

	// Various fixups
	if h.info.Config.UID != "" && h.info.Config.GID == "" {
		gid, err := passwd.GetGIDForUID(h.info.Config.UID)
		if err != nil {
			return err
		}
		h.info.Config.GID = strconv.FormatInt(int64(gid), 10)
	}

	if h.info.DefaultChroot == "" {
		h.info.DefaultChroot = "/"
	}

	chrootPath := h.info.Config.Chroot
	if chrootPath == "" {
		chrootPath = h.info.DefaultChroot
	}

	uid := -1
	gid := -1
	if h.info.Config.UID != "" {
		var err error
		uid, err = passwd.ParseUID(h.info.Config.UID)
		if err != nil {
			return err
		}

		gid, err = passwd.ParseGID(h.info.Config.GID)
		if err != nil {
			return err
		}
	}

	if (uid <= 0) != (gid <= 0) {
		return fmt.Errorf("Either both or neither of the UID and GID must be positive")
	}

	if uid > 0 {
		chrootErr, err := daemon.DropPrivileges(uid, gid, chrootPath)
		if err != nil {
			return fmt.Errorf("Failed to drop privileges: %v", err)
		}
		if chrootErr != nil && h.info.Config.Chroot != "" && h.info.Config.Chroot != "/" {
			return fmt.Errorf("Failed to chroot: %v", chrootErr)
		}
	} else if h.info.Config.Chroot != "" && h.info.Config.Chroot != "/" {
		return fmt.Errorf("Must use privilege dropping to use chroot; set -uid")
	}

	// If we still have any caps (maybe because we didn't setuid), try and drop them.
	err := caps.Drop()
	if err != nil {
		return fmt.Errorf("cannot drop caps: %v", err)
	}

	if !h.info.AllowRoot && daemon.IsRoot() {
		return fmt.Errorf("Daemon must not run as root or with capabilities; run as non-root user or use -uid")
	}

	h.dropped = true
	return nil
}
