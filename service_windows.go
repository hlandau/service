package service

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
	"gopkg.in/hlandau/svcutils.v1/exepath"
)

// This is always empty on Windows, as Windows does not support chrooting.
// It is present to allow code relying upon it to compile upon all platforms.
var EmptyChrootPath = ""

var errNotSupported = fmt.Errorf("not supported")

func systemdUpdateStatus(status string) error {
	return errNotSupported
}

func usingPlatform(platformName string) bool {
	return platformName == "windows"
}

// handler is used when running as a service.
// Otherwise we use the generic ihandler.
type handler struct {
	info        *Info
	startedChan chan struct{}
	stopChan    chan struct{}
	status      string
	dropped     bool
}

func (h *handler) DropPrivileges() error {
	h.dropped = true
	return nil
}

func (h *ihandler) DropPrivileges() error {
	h.dropped = true
	return nil
}

func (h *handler) SetStarted() {
	if !h.dropped {
		panic("service must call DropPrivileges before calling SetStarted")
	}

	select {
	case h.startedChan <- struct{}{}:
	default:
	}
}

func (h *handler) StopChan() <-chan struct{} {
	return h.stopChan
}

func (h *handler) SetStatus(status string) {
	h.status = status
}

func (h *handler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	h.startedChan = make(chan struct{}, 1)
	h.stopChan = make(chan struct{})
	doneChan := make(chan error)
	started := false
	stopping := false

	go func() {
		err := h.info.RunFunc(h)
		doneChan <- err
	}()

	var err error

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus

			case svc.Stop, svc.Shutdown:
				// Service stop is pending. Don't accept any more commands while pending.
				changes <- svc.Status{State: svc.StopPending}
				if !stopping {
					stopping = true
					close(h.stopChan)
				}

			default:
				// Unexpected control request
			}

		case <-h.startedChan:
			if started {
				panic("must not call SetStarted() more than once")
			}
			started = true
			changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

		case err = <-doneChan:
			break loop
		}
	}

	if err == nil {
		changes <- svc.Status{State: svc.Stopped}
		return false, 0
	} else {
		return false, 1
	}
}

func isInteractive() bool {
	interactive, err := svc.IsAnInteractiveSession()
	if err != nil {
		return false
	}
	return interactive
}

func (info *Info) installService() error {
	svcName := info.Name

	// Connect to the Windows service manager.
	serviceManager, err := mgr.Connect()
	if err != nil {
		return err
	}

	defer serviceManager.Disconnect()

	// Ensure the service doesn't already exist.
	service, err := serviceManager.OpenService(svcName)
	if err == nil {
		service.Close()
		return fmt.Errorf("service %s already exists", svcName)
	}

	// Install the service.
	service, err = serviceManager.CreateService(svcName, exepath.Abs, mgr.Config{
		DisplayName:  info.Title,
		Description:  info.Description,
		StartType:    mgr.StartAutomatic,
		ErrorControl: mgr.ErrorNormal,
	})
	if err != nil {
		return err
	}
	defer service.Close()

	// TODO: event log

	return nil
}

func (info *Info) removeService() error {
	svcName := info.Name

	// Connect to the Windows service manager.
	serviceManager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer serviceManager.Disconnect()

	// Ensure the service exists.
	service, err := serviceManager.OpenService(svcName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", svcName)
	}
	defer service.Close()

	// Remove the service.
	err = service.Delete()
	if err != nil {
		return err
	}

	return nil
}

func (info *Info) startService() error {
	svcName := info.Name

	// Connect to the Windows service manager.
	serviceManager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer serviceManager.Disconnect()

	service, err := serviceManager.OpenService(svcName)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer service.Close()

	err = service.Start(os.Args...)
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}

	return nil
}

func (info *Info) controlService(c svc.Cmd, to svc.State) error {
	svcName := info.Name

	// Connect to the Windows service manager.
	serviceManager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer serviceManager.Disconnect()

	service, err := serviceManager.OpenService(svcName)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer service.Close()

	// Send the control message.
	status, err := service.Control(c)
	if err != nil {
		return fmt.Errorf("could not send control=%d: %v", c, err)
	}

	// Wait.
	for status.State != to {
		time.Sleep(300 * time.Millisecond)
		status, err = service.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}

	return nil
}

func (info *Info) stopService() error {
	return info.controlService(svc.Stop, svc.Stopped)
}

func (info *Info) runAsService() error {
	// TODO: event log

	err := svc.Run(info.Name, &handler{info: info})
	if err != nil {
		return err
	}

	return nil
}

func (info *Info) serviceMain() error {
	switch info.Config.Command {
	case "install":
		return info.installService()
	case "remove":
		return info.removeService()
	case "start":
		return info.startService()
	case "stop":
		return info.stopService()
	default:
		// ...
	}

	interactive := isInteractive()
	if !interactive {
		return info.runAsService()
	}

	return info.runInteractively()
}

// Copyright © 2013-2014 Conformal Systems LLC.
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
