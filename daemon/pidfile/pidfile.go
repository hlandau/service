package pidfile

import "fmt"
import "os"
import "syscall"

var pidFile *os.File
var pidFileName string

// Opens a PID file. The PID of the current process is written to the file
// and the file is locked. The file is kept open until the process terminates.
// Only one PID file must be called at a time, so this function must not be
// called again unless ClosePIDFile() has been called.
func OpenPIDFile(filename string) error {
	if pidFile != nil {
		return fmt.Errorf("PID file already opened")
	}

	f, err := openPIDFile(filename)
	if err != nil {
		return err
	}

	s := fmt.Sprintf("%d\n", os.Getpid())
	_, err = f.WriteString(s)
	if err != nil {
		f.Close()
		return err
	}

	pidFile = f
	pidFileName = filename

	return nil
}

// Closes any previously opened PID file.
func ClosePIDFile() {
	if pidFile != nil {
		// try and remove file, don't care if it fails
		os.Remove(pidFileName)

		pidFile.Close()
		pidFile = nil
		pidFileName = ""
	}
}

func openPIDFile(filename string) (*os.File, error) {
	var f *os.File
	var err error

	for {
		f, err = os.OpenFile(filename,
			syscall.O_RDWR|syscall.O_CREAT|syscall.O_EXCL, 420 /* 0644 */)
		if err != nil {
			if !os.IsExist(err) {
				return nil, err
			}

			f, err = os.OpenFile(filename, syscall.O_RDWR, 420 /* 0644 */)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
		}

		err = syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, &syscall.Flock_t{
			Type: syscall.F_WRLCK,
		})
		if err != nil {
			f.Close()
			return nil, err
		}

		st1 := syscall.Stat_t{}
		err = syscall.Fstat(int(f.Fd()), &st1) // ffs
		if err != nil {
			f.Close()
			return nil, err
		}

		st2 := syscall.Stat_t{}
		err = syscall.Stat(filename, &st2)
		if err != nil {
			f.Close()

			if os.IsNotExist(err) {
				continue
			}

			return nil, err
		}

		if st1.Ino != st2.Ino {
			f.Close()
			continue
		}

		break
	}

	return f, nil
}
