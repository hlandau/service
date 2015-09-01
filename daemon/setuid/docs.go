// Package setuid provides functions to change the current UID and GID on *nix systems.
//
// This is somewhat harder than it seems. The syscall package provides Setuid,
// etc., but on Linux, these functions are a trap:
//
// Linux has a faulty, non-compliant implementation of setuid(2) which only
// changes the UID of the current thread, not the whole process. Amazingly,
// even the manual page lies and claims that it affects the process.
//
// glibc's setuid syscall wrapper dispatches setuid calls to all threads. Ergo,
// the manual page for setuid(3) but not setuid(2) is accurate.
//
// The same also applies to setgid, setresuid, setresgid, etc. Though oddly
// enough not setgroups.
//
// Therefore setuid, setgid, setresuid and setresgid are dispatched through
// cgo, hence this package rather than using the syscall package.
//
// These functions are only available on *NIX platforms.
package setuid
