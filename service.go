// Package service wraps all the complexity of writing daemons while enabling
// seamless integration with OS service management facilities.
//
// # Changes in v3
//
// v2 of this package used [configurable] and [easyconfig] to configure
// service-related parameters. This allowed other packages to automatically
// discover global configurable dials for any package linked into a Go binary
// and expose them as command line arguments. This approach has been deprecated
// as it exposes internal details of an application's organisation as part of its
// CLI interface and makes it hard to maintain over time. This approach is
// deprecated in favour of an explicit approach where service variables are
// specified in a structure, [Config].
//
// v3 no longer links to [easyconfig], reducing its dependency closure size,
// and instead simply accepts a mandatory Config structure which can be used to
// specify the configuration parameters for a service.
//
// v3 removes support for launching a debug HTTP server. An application can provide
// this functionality itself if needed. This reduces dependency closure size by allowing
// this package to no longer depend on net/http.
//
// # Platform-Specific Configuration Variables
//
// Some fields in [Config] are platform-specific. The fields are present on all
// platforms as Go provides no simple way to omit fields in structure
// definitions on certain platforms. The "platform" annotation on a field
// denotes if a field is platform-specific. If this annotation is omitted, the
// field is supported on all platforms. You can pass the "platform" annotation
// to [UsingPlatform] to determine if a field is currently applicable.
//
// [configurable]: https://github.com/hlandau/configurable
// [easyconfig]: https://github.com/hlandau/easyconfig
package service // import "gopkg.in/hlandau/service.v3"

import (
	"expvar"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"

	"gopkg.in/hlandau/service.v3/gsptcall"
	"gopkg.in/hlandau/svcutils.v1/exepath"
)

type nullWriter struct{}

func (nw nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func init() {
	expvar.NewString("service.startTime").Set(time.Now().String())
}

// This function should typically be called directly from func main(). It takes
// care of all housekeeping for running services and handles service lifecycle.
func Main(info *Info) {
	info.main()
}

// The interface between the service library and the application-specific code.
// The application calls the methods in the provided instance of this interface
// at various stages in its lifecycle.
type Manager interface {
	// Must be called when the service is ready to drop privileges.
	// This must be called before SetStarted().
	DropPrivileges() error

	// Must be called by a service payload when it has finished starting.
	SetStarted()

	// A service payload must stop when this channel is closed.
	StopChan() <-chan struct{}

	// Called by a service payload to provide a single line of information on the
	// current status of that service.
	SetStatus(status string)
}

// Used only by the NewFunc interface.
type Runnable interface {
	// Start the runnable. Any initialization requiring root privileges must
	// already have been obtained as this will be called after dropping
	// privileges. Must return.
	Start() error

	// Stop the runnable. Must return.
	Stop() error
}

// An upgrade interface for Runnable, implementation of which is optional.
type StatusSource interface {
	// Return a channel on which status messages will be sent. If a Runnable
	// implements this, it is guaranteed that the channel will be consumed until
	// Stop is called.
	StatusChan() <-chan string
}

// Configuration variables which control how a service is run.
type Config struct {
	// If this is non-empty, CPU profiling is initiated on startup and the
	// profile is written to the given file.
	CPUProfile string `help:"Write CPU profile to file"`

	// UNIX: If this is non-empty, privilege dropping is enabled. The value can be a UID or username.
	UID string `help:"UID to run as (default: don't drop privileges)" platform:"unix"`

	// UNIX: If this is non-empty, it is the GID or group name used when dropping
	// privileges. If privilege dropping is enabled (UID is non-empty) and this
	// is empty, the GID for the given UID is looked up from the system.
	GID string `help:"GID to run as (default: don't drop privileges)" platform:"unix"`

	// UNIX: Runs the service as a daemon (aside from forking). This sets up the
	// CWD, umask, calls setsid() and remaps stdin and stdout (and stderr, if
	// Stderr is not set) to /dev/null.
	Daemon bool `help:"Run as daemon? (doesn't fork)" platform:"unix"`

	// UNIX: Fork. Implies Daemon.
	Fork bool `help:"Fork? (implies daemon)" platform:"unix"`

	// UNIX: If non-empty, path to a file to write the process PID to.
	PIDFile string `help:"Write PID to file with given filename and hold a write lock" platform:"unix"`

	// UNIX: If not "/", the directory to chroot into. Only used if dropping
	// privileges (i.e., if UID is non-empty).
	Chroot string `help:"Chroot to a directory (must set UID, GID) ('/' disables)" platform:"unix"`

	// UNIX: Keep stderr open if Daemon is set and do not remap it to /dev/null.
	Stderr bool `help:"Keep stderr open when daemonizing" platform:"unix"`

	// Windows: Service control command. Can be used to install or uninstall a
	// service, or start or stop it. If empty, run the service normally.
	// The package automatically detects if it is running under the service manager
	// or as a normal process.
	Command string `help:"Service command (install, uninstall, start, stop)" platform:"windows"`
}

// Returns true if a given platform name (e.g. "", "unix", "windows") is currently applicable.
func UsingPlatform(platformName string) bool {
	if platformName == "" {
		return true
	}
	return usingPlatform(platformName)
}

// An instantiable service.
type Info struct {
	// Recommended. Codename for the service, e.g. "foobar"
	//
	// If this is not set, exepath.ProgramName is used, which by default is the
	// program's binary basename (e.g. "FooBar.exe" would become "foobar").
	Name string

	// Required unless NewFunc is specified instead. Starts the service. Must not
	// return until the service has stopped. Must call smgr.SetStarted() to
	// indicate when it has finished starting and use smgr.StopChan() to
	// determine when to stop.
	//
	// Should call SetStatus() periodically with a status string.
	RunFunc func(smgr Manager) error

	// Optional. An alternative to RunFunc. If this is provided, RunFunc must not
	// be specified, and this package will provide its own implementation of
	// RunFunc.
	//
	// The NewFunc will be called to instantiate the runnable service.
	// Privileges will then be dropped and Start will be called. Start must
	// return. When the service is to be stopped, Stop will be called. Stop must
	// return.
	//
	// To implement status notification, implement also the StatusSource interface.
	NewFunc func() (Runnable, error)

	Title       string // Optional. Friendly name for the service, e.g. "Foobar Web Server"
	Description string // Optional. Single line description for the service

	AllowRoot     bool   // May the service run as root? If false, the service will refuse to run as root unless privilege dropping is set.
	DefaultChroot string // Default path to chroot to. Use this if the service can be chrooted without consequence.
	NoBanSuid     bool   // Set to true if the ability to execute suid binaries must be retained.

	// This must contain the configuration variables to be used to run the service. It will generally be parsed by an application from a command line.
	Config Config

	// Are we being started by systemd with [Service] Type=notify?
	// If so, we can issue service status notifications to systemd.
	systemd bool

	// Path to created PID file.
	pidFileName string
	pidFile     io.Closer
}

func (info *Info) main() {
	err := info.maine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in service: %+v\n", err)
		os.Exit(1)
	}
}

func (info *Info) maine() error {
	if info.Name == "" {
		info.Name = exepath.ProgramName
	} else if exepath.ProgramNameSetter == "default" {
		exepath.ProgramName = info.Name
		exepath.ProgramNameSetter = "service"
	}

	if info.Name == "" {
		panic("service name must be specified")
	}
	if info.Title == "" {
		info.Title = info.Name
	}
	if info.Description == "" {
		info.Description = info.Title
	}

	err := info.commonPre()
	if err != nil {
		return err
	}

	err = info.setRunFunc()
	if err != nil {
		return err
	}

	// profiling
	if info.Config.CPUProfile != "" {
		f, err := os.Create(info.Config.CPUProfile)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer f.Close()
		defer pprof.StopCPUProfile()
	}

	err = info.serviceMain()

	return err
}

func (info *Info) commonPre() error {
	return nil
}

func (info *Info) setRunFunc() error {
	if info.RunFunc != nil {
		return nil
	}

	if info.NewFunc == nil {
		panic("either RunFunc or NewFunc must be specified")
	}

	info.RunFunc = func(smgr Manager) error {
		// instantiate runnable
		r, err := info.NewFunc()
		if err != nil {
			return err
		}

		// setup status channel
		getStatusChan := func() <-chan string {
			return nil
		}
		if ss, ok := r.(StatusSource); ok {
			getStatusChan = func() <-chan string {
				return ss.StatusChan()
			}
		}

		// drop privileges
		err = smgr.DropPrivileges()
		if err != nil {
			return err
		}

		// start
		err = r.Start()
		if err != nil {
			return err
		}

		//
		smgr.SetStarted()
		smgr.SetStatus(info.Name + ": running ok")

		// wait for status messages or stop requests
	loop:
		for {
			select {
			case statusMsg := <-getStatusChan():
				smgr.SetStatus(info.Name + ": " + statusMsg)

			case <-smgr.StopChan():
				break loop
			}
		}

		// stop
		return r.Stop()
	}

	return nil
}

type ihandler struct {
	info             *Info
	stopChan         chan struct{}
	statusMutex      sync.Mutex
	statusNotifyChan chan struct{}
	startedChan      chan struct{}
	status           string
	started          bool
	stopping         bool
	dropped          bool
}

func (h *ihandler) SetStarted() {
	if !h.dropped {
		panic("service must call DropPrivileges before calling SetStarted")
	}

	select {
	case h.startedChan <- struct{}{}:
	default:
	}
}

func (h *ihandler) StopChan() <-chan struct{} {
	return h.stopChan
}

func (h *ihandler) SetStatus(status string) {
	h.statusMutex.Lock()
	h.status = status
	h.statusMutex.Unlock()

	select {
	case <-h.statusNotifyChan:
	default:
	}
}

func (h *ihandler) updateStatus() {
	// systemd
	if h.info.systemd {
		s := ""
		if h.started {
			s += "READY=1\n"
		}
		if h.status != "" {
			s += "STATUS=" + h.status + "\n"
		}
		systemdUpdateStatus(s)
		// ignore error
	}

	if h.status != "" {
		gsptcall.SetProcTitle(h.status)
	}
}

func (info *Info) runInteractively() error {
	smgr := ihandler{
		info:             info,
		stopChan:         make(chan struct{}),
		statusNotifyChan: make(chan struct{}, 1),
		startedChan:      make(chan struct{}, 1),
	}

	doneChan := make(chan error)
	go func() {
		err := info.RunFunc(&smgr)
		doneChan <- err
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	var exitErr error

loop:
	for {
		select {
		case <-sig:
			if !smgr.stopping {
				smgr.stopping = true
				close(smgr.stopChan)
				smgr.updateStatus()
			}
		case <-smgr.startedChan:
			if !smgr.started {
				smgr.started = true
				smgr.updateStatus()
			}
		case <-smgr.statusNotifyChan:
			smgr.updateStatus()
		case exitErr = <-doneChan:
			break loop
		}
	}

	return exitErr
}
