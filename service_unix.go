package service

import "github.com/hlandau/service/passwd"
import "github.com/hlandau/degoutils/log"
import "github.com/hlandau/service/daemon"
import "github.com/hlandau/service/daemon/pidfile"
import "github.com/hlandau/service/sdnotify"
import "github.com/ErikDubbelboer/gspt"
import "os"
import "fmt"
import "flag"
import "strconv"

var uidFlag = fs.String("uid", "", "UID to run as (default: don't drop privileges)")
var _uidFlag = flag.String("uid", "", "UID to run as (default: don't drop privileges)")
var gidFlag = fs.String("gid", "", "GID to run as (default: don't drop privileges)")
var _gidFlag = flag.String("gid", "", "GID to run as (default: don't drop privileges)")
var daemonizeFlag = fs.Bool("daemon", false, "Run as daemon? (doesn't fork)")
var _daemonizeFlag = flag.Bool("daemon", false, "Run as daemon? (doesn't fork)")
var chrootFlag = fs.String("chroot", "", "Chroot to a directory (must set UID, GID) (\"/\" disables)")
var _chrootFlag = flag.String("chroot", "", "Chroot to a directory (must set UID, GID) (\"/\" disables)")
var pidfileFlag = fs.String("pidfile", "", "Write PID to file with given filename and hold a write lock")
var _pidfileFlag = flag.String("pidfile", "", "Write PID to file with given filename and hold a write lock")
var dropprivsFlag = fs.Bool("dropprivs", true, "Drop privileges?")
var _dropprivsFlag = flag.Bool("dropprivs", true, "Drop privileges?")
var forkFlag = fs.Bool("fork", false, "Fork? (implies -daemon)")
var _forkFlag = flag.Bool("fork", false, "Fork? (implies -daemon)")

func systemdUpdateStatus(status string) error {
	return sdnotify.SdNotify(status)
}

func setproctitle(status string) error {
	gspt.SetProcTitle(status)
	return nil
}

func (info *Info) serviceMain() error {
	err := daemon.Init()
	if err != nil {
		return err
	}

	err = systemdUpdateStatus("\n")
	if err == nil {
		info.systemd = true
	}

	if *pidfileFlag != "" {
		info.pidFileName = *pidfileFlag

		err = info.openPIDFile()
		if err != nil {
			return err
		}

		defer info.closePIDFile()
	}

	return info.runInteractively()
}

func (info *Info) openPIDFile() error {
	return pidfile.OpenPIDFile(info.pidFileName)
}

func (info *Info) closePIDFile() {
	pidfile.ClosePIDFile()
}

func (h *ihandler) DropPrivileges() error {
	if h.dropped {
		return nil
	}

	if *forkFlag {
		isParent, err := daemon.Fork()
		if err != nil {
			return err
		}

		if isParent {
			os.Exit(0)
		}

		*daemonizeFlag = true
	}

	if *daemonizeFlag || h.info.systemd {
		err := daemon.Daemonize()
		if err != nil {
			return err
		}
	}

	if *uidFlag != "" && *gidFlag == "" {
		gid, err := passwd.GetGIDForUID(*uidFlag)
		if err != nil {
			return err
		}
		*gidFlag = strconv.FormatInt(int64(gid), 10)
	}

	if h.info.DefaultChroot == "" {
		h.info.DefaultChroot = "/"
	}

	chrootPath := *chrootFlag
	if chrootPath == "" {
		chrootPath = h.info.DefaultChroot
	}

	uid := -1
	gid := -1
	if *uidFlag != "" {
		var err error
		uid, err = passwd.ParseUID(*uidFlag)
		if err != nil {
			return err
		}

		gid, err = passwd.ParseGID(*gidFlag)
		if err != nil {
			return err
		}
	}

	if *dropprivsFlag {
		chrootErr, err := daemon.DropPrivileges(uid, gid, chrootPath)
		if err != nil {
			log.Errore(err, "cannot drop privileges")
			return err
		}
		if chrootErr != nil && *chrootFlag != "" && *chrootFlag != "/" {
			return fmt.Errorf("Failed to chroot: %v", chrootErr)
		}
	} else if *chrootFlag != "" && *chrootFlag != "/" {
		return fmt.Errorf("Must set dropprivs to use chroot")
	}

	if !h.info.AllowRoot && daemon.IsRoot() {
		return fmt.Errorf("Daemon must not run as root or with capabilities")
	}

	h.dropped = true
	return nil
}
