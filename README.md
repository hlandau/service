service: Write daemons in Go
----------------------------

[![GoDoc](https://godoc.org/github.com/hlandau/service?status.svg)](https://godoc.org/github.com/hlandau/service)

This package enables you to easily write services in Go such that the following concerns are taken care of automatically:

  - Daemonization
  - Fork emulation (not recommended, though)
  - PID file creation
  - Privilege dropping
  - Chrooting
  - Status notification (supports setproctitle and systemd notify protocol; this support is Go-native and does not introduce any dependency on any systemd library)
  - Operation as a Windows service
  - Orderly shutdown

Here's a usage example:

    import "github.com/hlandau/service"

    func main() {
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

              // Wait until stop is requested.
              <-smgr.StopChan()

              // Do any necessary teardown.
              // ...

              // Done.
              return nil
          },
      })
    }
