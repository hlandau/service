package exepath

import (
	"os"
	"path/filepath"
)

// Absolute path to EXE which was invoked. This is set at init()-time
// to ensure that argv[0] can be properly interpreted before chdir is called.
var Abs string

func init() {
	Abs = os.Args[0]
	dir, err := filepath.Abs(Abs)
	if err != nil {
		return
	}

	Abs = dir
}
