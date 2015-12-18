// Package service wraps all the complexity of writing daemons while enabling
// seamless integration with OS service management facilities.
package service // import "gopkg.in/hlandau/service.v2"

import (
	"expvar"
	"fmt"
	"gopkg.in/hlandau/easyconfig.v1/cflag"
	"gopkg.in/hlandau/svcutils.v1/exepath"
	"net/http"
	_ "net/http/pprof" // register pprof handler for debug server
	"os"
	"os/signal"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"
)

// Flags

var (
	fg                  = cflag.NewGroup(nil, "service")
	cpuProfileFlag      = cflag.String(fg, "cpuprofile", "", "Write CPU profile to file")
	debugServerAddrFlag = cflag.String(fg, "debugserveraddr", "", "Address for debug server to listen on (do not specify a public address) (default: disabled)")
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

	// Are we being started by systemd with [Service] Type=notify?
	// If so, we can issue service status notifications to systemd.
	systemd bool

	// Path to created PID file.
	pidFileName string
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
	if cpuProfileFlag.Value() != "" {
		f, err := os.Create(cpuProfileFlag.Value())
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
	if debugServerAddrFlag != nil && debugServerAddrFlag.Value() != "" {
		go func() {
			err := http.ListenAndServe(debugServerAddrFlag.Value(), nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't start debug server: %+v\n", err)
			}
		}()
	}
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
		setproctitle(h.status)
		// ignore error
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

	sig := make(chan os.Signal)
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

// © 2015 Hugo Landau <hlandau@devever.net>  ISC License
