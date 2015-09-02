package exepath

import (
	"os"
	"path/filepath"
)

// Absolute path to EXE which was invoked. This is set at init()-time.
var Abs string

func getRawPath() string {
	// "github.com/kardianos/osext".Executable looks nice, but may return the
	// realpath of the path because this is how the kernel returns it as
	// /proc/self/exe. This causes problems with layouts like
	//
	//   some-work-directory/
	//     bin/ -> $GOPATH/bin
	//     src/ -> $GOPATH/bin
	//     etc/
	//       ... configuration files ...
	//
	// where bin/foo is executed from some-work-directory and expects to find files in etc/.
	// Since -fork reexecutes with the exepath.Abs path, this prevents paths like
	//   $BIN/../etc/foo.conf from working (where $BIN is the dir of the executable path).
	//
	// Okay, maybe this is a byzantine configuration. But still, this breaks my existing
	// configuration, so I'm sticking with os.Args[0] for now, as -fork should be as seamless
	// as possible to relying applications.

	return os.Args[0]
}

func init() {
	abs, err := filepath.Abs(getRawPath())
	if err != nil {
		return
	}

	Abs = abs
}
