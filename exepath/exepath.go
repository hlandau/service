package exepath

import "os"
import "path/filepath"

// Absolute path to EXE which was invoked. This is set at init()-time
// to ensure that argv[0] can be properly interpreted before chdir is called.
var AbsExePath string

func init() {
	AbsExePath = os.Args[0]
	dir, err := filepath.Abs(AbsExePath)
	if err != nil {
		return
	}

	AbsExePath = dir
}
