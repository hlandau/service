// +build !windows,!plan9

package dupfd

func Dup2(sourceFD, targetFD int) error {
	return dup2(sourceFD, targetFD)
}
