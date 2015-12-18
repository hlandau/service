// +build !windows

package service

import (
	"fmt"
	"github.com/hlandauf/gspt"
	"gopkg.in/hlandau/easyconfig.v1/cflag"
	"gopkg.in/hlandau/service.v2/daemon"
	"gopkg.in/hlandau/service.v2/daemon/bansuid"
	"gopkg.in/hlandau/service.v2/daemon/caps"
	"gopkg.in/hlandau/service.v2/daemon/pidfile"
	"gopkg.in/hlandau/service.v2/passwd"
	"gopkg.in/hlandau/svcutils.v1/systemd"
	"os"
	"strconv"
)

// This will always point to a path which the platform guarantees is an empty
// directory. You can use it as your default chroot path if your service doesn't
// access the filesystem after it's started.
//
// On Linux, the FHS provides that "/var/empty" is an empty directory, so it
// points to that.
var EmptyChrootPath = daemon.EmptyChrootPath

var (
	uidFlag       = cflag.String(fg, "uid", "", "UID to run as (default: don't drop privileges)")
	gidFlag       = cflag.String(fg, "gid", "", "GID to run as (default: don't drop privileges)")
	daemonizeFlag = cflag.Bool(fg, "daemon", false, "Run as daemon? (doesn't fork)")
	chrootFlag    = cflag.String(fg, "chroot", "", "Chroot to a directory (must set UID, GID) (\"/\" disables)")
	pidfileFlag   = cflag.String(fg, "pidfile", "", "Write PID to file with given filename and hold a write lock")
	forkFlag      = cflag.Bool(fg, "fork", false, "Fork? (implies -daemon)")
)

func systemdUpdateStatus(status string) error {
	return systemd.NotifySend(status)
}

func setproctitle(status string) error {
	gspt.SetProcTitle(status)
	return nil
}

func (info *Info) serviceMain() error {
	if forkFlag.Value() {
		isParent, err := daemon.Fork()
		if err != nil {
			return err
		}

		if isParent {
			os.Exit(0)
		}

		daemonizeFlag.SetValue(true)
	}

	err := daemon.Init()
	if err != nil {
		return err
	}

	err = systemdUpdateStatus("\n")
	if err == nil {
		info.systemd = true
	}

	if daemonizeFlag.Value() || info.systemd {
		err := daemon.Daemonize()
		if err != nil {
			return err
		}
	}

	if pidfileFlag.Value() != "" {
		info.pidFileName = pidfileFlag.Value()

		err = info.openPIDFile()
		if err != nil {
			return err
		}

		defer info.closePIDFile()
	}

	return info.runInteractively()
}

func (info *Info) openPIDFile() error {
	return pidfile.Open(info.pidFileName)
}

func (info *Info) closePIDFile() {
	pidfile.Close()
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
	if uidFlag.Value() != "" && gidFlag.Value() == "" {
		gid, err := passwd.GetGIDForUID(uidFlag.Value())
		if err != nil {
			return err
		}
		gidFlag.SetValue(strconv.FormatInt(int64(gid), 10))
	}

	if h.info.DefaultChroot == "" {
		h.info.DefaultChroot = "/"
	}

	chrootPath := chrootFlag.Value()
	if chrootPath == "" {
		chrootPath = h.info.DefaultChroot
	}

	uid := -1
	gid := -1
	if uidFlag.Value() != "" {
		var err error
		uid, err = passwd.ParseUID(uidFlag.Value())
		if err != nil {
			return err
		}

		gid, err = passwd.ParseGID(gidFlag.Value())
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
		if chrootErr != nil && chrootFlag.Value() != "" && chrootFlag.Value() != "/" {
			return fmt.Errorf("Failed to chroot: %v", chrootErr)
		}
	} else if chrootFlag.Value() != "" && chrootFlag.Value() != "/" {
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

// © 2015 Hugo Landau <hlandau@devever.net>  ISC License
