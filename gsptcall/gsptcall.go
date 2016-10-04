// Package gsptcall provides a call wrapper for SetProcTitle which does nothing
// when it is not supported.
package gsptcall

// Calls erikdubbelboer/gspt.SetProcTitle, but only on UNIX platforms and where
// cgo is enabled. Otherwise, it is a no-op.
func SetProcTitle(title string) {
	setProcTitle(title)
}
