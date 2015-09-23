// +build !windows

// Package daemon provides functions to assist with the writing of UNIX-style
// daemons in go.
package daemon

import (
	"gopkg.in/hlandau/svcutils.v1/exepath"
	"os"
	"syscall"
)

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
	newArgs = append(newArgs, exepath.Abs)
	newArgs = append(newArgs, os.Args[1:]...)
	newArgs = append(newArgs, forkedArg)

	// Start the child process.
	//
	// Pass along the standard FD for now - we'll remap them to /dev/null
	// in due time. This ensures anything expecting these to exist isn't confused,
	// and allows pre-daemonization failures to at least get output to somewhere.
	proc, err := os.StartProcess(exepath.Abs, newArgs, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
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
