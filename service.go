// Package service wraps all the complexity of writing daemons while enabling
// seamless integration with OS service management facilities.
package service // import "gopkg.in/hlandau/service.v1"

import (
	"expvar"
	"flag"
	"fmt"
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
	fs                   = flag.NewFlagSet("Service Options", flag.ContinueOnError)
	debugServerAddrFlag  *string
	_debugServerAddrFlag *string
	cpuProfileFlag       = fs.String("cpuprofile", "", "Write CPU profile to file")
	_cpuProfileFlag      = flag.String("cpuprofile", "", "Write CPU profile to file")
)

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

// An instantiable service.
type Info struct {
	Name string // Required. Codename for the service, e.g. "foobar"

	// Required. Starts the service. Must not return until the service has
	// stopped. Must call smgr.SetStarted() to indicate when it has finished
	// starting and use smgr.StopChan() to determine when to stop.
	//
	// Should call SetStatus() periodically with a status string.
	RunFunc func(smgr Manager) error

	Title       string // Optional. Friendly name for the service, e.g. "Foobar Web Server"
	Description string // Optional. Single line description for the service

	AllowRoot     bool   // May the service run as root? If false, the service will refuse to run as root unless privilege dropping is set.
	DefaultChroot string // Default path to chroot to. Use this if the service can be chrooted without consequence.
	NoBanSuid     bool   // Set to true if the ability to execute suid binaries must be retained.

	// Set to true if the service uses the default HTTP serve mux. This disables
	// the -debugserveraddr option.
	UsesDefaultHTTP bool

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
		panic("service name must be specified")
	}
	if info.Title == "" {
		info.Title = info.Name
	}
	if info.Description == "" {
		info.Description = info.Title
	}

	if !info.UsesDefaultHTTP {
		debugServerAddrFlag = fs.String("debugserveraddr", "", "Address for debug server to listen on (do not specify a public address) (default: disabled)")
		_debugServerAddrFlag = flag.String("debugserveraddr", "", "Address for debug server to listen on (do not specify a public address) (default: disabled)")
	}

	fs.Parse(os.Args[1:]) // ignore errors

	err := info.commonPre()
	if err != nil {
		return err
	}

	// profiling
	if *cpuProfileFlag != "" {
		f, err := os.Create(*cpuProfileFlag)
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
	if debugServerAddrFlag != nil && *debugServerAddrFlag != "" {
		go func() {
			err := http.ListenAndServe(*debugServerAddrFlag, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't start debug server: %+v\n", err)
			}
		}()
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
	smgr := ihandler{info: info}
	smgr.stopChan = make(chan struct{})
	smgr.statusNotifyChan = make(chan struct{}, 1)
	smgr.startedChan = make(chan struct{}, 1)

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
