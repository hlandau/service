package service_test

import "gopkg.in/hlandau/service.v1"

// The following example illustrates the minimal skeleton structure to
// implement a daemon. This example can run as a service on Windows or a daemon
// on Linux.  The systemd notify protocol is supported.
func Example() {
	service.Main(&service.Info{
		Title:       "Foobar Web Server",
		Name:        "foobar",
		Description: "Foobar Web Server is the greatest webserver ever.",

		RunFunc: func(smgr service.Manager) error {
			// Start up your service.
			// ...

			// Once initialization requiring root is done, call this.
			err := smgr.DropPrivileges()
			if err != nil {
				return err
			}

			// When it is ready to serve requests, call this.
			// You must call DropPrivileges first.
			smgr.SetStarted()

			// Optionally set a status
			smgr.SetStatus("foobar: running ok")

		loop:
			for {
				select {
				// Handle requests, or do so in another goroutine controlled from here.
				case <-smgr.StopChan():
					break loop
				}
			}

			// Do any necessary teardown.
			// ...

			return nil
		},
	})
}

// © 2015 Hugo Landau <hlandau@devever.net>  ISC License
