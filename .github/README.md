service: Write daemons in Go
============================

[![godocs.io](https://godocs.io/gopkg.in/hlandau/service.v2?status.svg)](https://godocs.io/gopkg.in/hlandau/service.v2) [![Build status](https://github.com/hlandau/service/actions/workflows/go.yml/badge.svg)](#) [![No modules](https://www.devever.net/~hl/f/no-modules2.svg) 100% modules-free.](https://www.devever.net/~hl/gomod)

This package enables you to easily write services in Go such that the following concerns are taken care of automatically:

  - Daemonization
  - Fork emulation (not recommended, though)
  - PID file creation
  - Privilege dropping
  - Chrooting
  - Status notification (supports setproctitle and systemd notify protocol; this support is Go-native and does not introduce any dependency on any systemd library)
  - Operation as a Windows service
  - Orderly shutdown

Standard Interface
------------------

Here's a usage example:

```go
package main

import "gopkg.in/hlandau/service.v2"
import "gopkg.in/hlandau/easyconfig.v1"

func main() {
  easyconfig.ParseFatal(nil, nil)

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

          // Optionally set a status.
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
```

You should import the package as "gopkg.in/hlandau/service.v2". Compatibility will be preserved. (Please note that this compatibility guarantee does not extend to subpackages.)

Simplified Interface
--------------------

If you implement the following interface, you can use the simplified interface. This example also demonstrates how to use easyconfig to handle your configuration.

```go
  func() (Runnable, error)

  type Runnable interface {
    Start() error
    Stop() error
  }
```

Usage example:

```go
package main

import "gopkg.in/hlandau/service.v2"
import "gopkg.in/hlandau/easyconfig.v1"

type Config struct{}

// Server which doesn't do anything
type Server struct{}

func New(cfg Config) (*Server, error) {
  // Instantiate the service and bind to ports here
  return &Server{}, nil
}

func (*Server) Start() error {
  // Start handling of requests here (must return)
  return nil
}

func (*Server) Stop() error {
  // Stop the service here
  return nil
}

func main() {
  cfg := Config{}

  configurator := easyconfig.Configurator{
    ProgramName: "foobar",
  }

  configurator.ParseFatal(&cfg)

  service.Main(&service.Info{
    Name:        "foobar",

    NewFunc: func() (service.Runnable, error) {
      return New(cfg)
    },
  })
}
```

Changes since v1
----------------

v1 used the "flag" package to register service configuration options like UID, GID, etc.

v2 uses the "[configurable](https://github.com/hlandau/configurable)" package
to register service configuration options. "configurable" is a neutral
[integration nexus](http://www.devever.net/~hl/nexuses), so it increases the
generality of `service`. However, bear in mind that you are responsible for
ensuring that configuration is loaded before calling service.Main.

Configurables
-------------

The following configurables are automatically registered under a group configurable named "service":

    chroot          (string) path       (*nix only) chroot to a directory (must set UID, GID) ("/" disables)
    daemon          (bool)              (*nix only) run as daemon? (doesn't fork)
                                        (remaps stdin, stdout, stderr to /dev/null; calls setsid)
    fork            (bool)              (*nix only) fork?
    uid             (string) username   (*nix only) UID or username to setuid to
    gid             (string) groupname  (*nix only) GID or group name to setgid to
    pidfile         (string) path       (*nix only) Path of PID file to write and lock (default: no PID file)

    do              (string) start|stop|install|remove  (Windows only) Service control.

    cpuprofile      (string) path       Write CPU profile to file
    debugserveraddr (string) ip:port    Bind the net/http DefaultServeMux to the given address
                                        (expvars, pprof handlers will be registered; intended for debug use only;
                                        set UsesDefaultHTTP in the Info type to disable the presence of this flag)

If you call `easyconfig.ParseFatal(nil, nil)` as suggested above, these manifest as "flag" flags named -service.X,
for each name X above. e.g. `-service.chroot=/`

Using as a Windows service
--------------------------

You can use the `-service.do=install` and `-service.do=remove` flags to install and
remove the service as a Windows service. Please note that:

  - You will need to run these commands from an elevated command prompt
    (right click on 'Command Prompt' and select 'Run as administrator').

  - The absolute path of the executable in its current location will be used
    as the path to the service.

  - You may need to tweak the command line arguments for the service
    to your liking using `services.msc` after installation.

  - You may also use any other method that you like to install or remove
    services. No particular command line flag is required; the service will
    detect when it is being run as a Windows service automatically.

### Manifests

If your service *always* needs to run privileged, you may want to apply a manifest file to your binary to make elevation automatic. You should avoid this if your service can be configured to usefully operate without elevation, as it denies the user choice in how to run the service.

Here is an example manifest:

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
    <trustInfo xmlns="urn:schemas-microsoft-com:asm.v2">
        <security>
            <requestedPrivileges>
                <requestedExecutionLevel 
                     level="requireAdministrator" 
                     uiAccess="false"/>
            </requestedPrivileges>
        </security>
    </trustInfo>
</assembly>
```

You can use this manifest either as a sidecar file by naming it `<exe-name>.exe.manifest`, or by embedding it into the binary. You may wish to investigate Microsoft's `mt` tool or [akavel/rsrc](https://github.com/akavel/rsrc), which provides a Go-specific solution.

For more information on manifests, see MSDN.

Use with systemd
----------------

Here is an example systemd unit file with privilege dropping and auto-restart:

    [Unit]
    Description=short description of the daemon
    ;; Optionally make the service dependent on other services
    ;Requires=other.service

    [Service]
    Type=notify
    ExecStart=/path/to/foobar/foobard -service.uid=foobar -service.gid=foobar -service.daemon=1
    Restart=always
    RestartSec=30

    [Install]
    WantedBy=multi-user.target

Bugs
----

  - If you don't consume registered configurables, the user cannot configure
    the options of this package, rendering it somewhat unusable. You must handle
    registered configurables. The easyconfig example above suffices.

  - Testing would be nice, but a library of this nature isn't too susceptible
    to unit testing. Something to think about.

  - **Severe**: A bug in Go 1.5 means that privilege dropping does not work correctly, but instead hangs forever ([#12498](https://github.com/golang/go/issues/12498)). A patch is available but is not yet part of any release. As a workaround, use Go 1.4 or do not use privilege dropping (e.g. run as a non-root user and do not specify `-uid`, `-gid` or `-chroot`). If you need to bind to low ports, you can use `setcap` on Linux to grant those privileges. (This bug is fixed in Go 1.5.2 and later.)

Platform Support
----------------

The package should work on Windows or any UNIX-like platform, but has been
tested only on the following platforms:

  - Linux
  - FreeBSD
  - Darwin/OS X
  - Windows

On Linux **you may need to install the libcap development package** (`libcap-dev` on Debian-style distros, `libcap-devel` on Red Hat-style distros), as this package uses libcap to make sure all capabilities are dropped on Linux.

Reduced Functionality Mode
--------------------------

When built without cgo, the following limitations are imposed:

  - Privilege dropping is not supported at all on Linux.
  - UIDs and GIDs must be specified numerically, not as names.
  - No supplementary GIDs are configured when dropping privileges (the empty set is configured).
  - setproctitle is not supported; status setting is a no-op.

Utility Library
---------------

This package provides a simplified interface built on some functionality
exposed in [hlandau/svcutils](https://github.com/hlandau/svcutils). People who
want something less “magic” may find functions there useful.

Some functions in that repository may still be useful to people using this
package. For example, the chroot package allows you to (try to) relativize a
path to a chroot, allowing you to address files by their absolute path after
chrooting.

Licence
-------

    ISC License

    Permission to use, copy, modify, and distribute this software for any
    purpose with or without fee is hereby granted, provided that the above
    copyright notice and this permission notice appear in all copies.

    THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
    WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
    MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
    ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
    WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
    ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
    OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

