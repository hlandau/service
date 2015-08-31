package daemon

import "syscall"
import "fmt"

func setSecurebits() error {
	err := prctl(pPR_SET_SECUREBITS,
		sSECBIT_NOROOT|sSECBIT_NOROOT_LOCKED|sSECBIT_KEEP_CAPS_LOCKED, 0, 0, 0)
	if err != nil {
		return fmt.Errorf("cannot set SECUREBITS: %v", err)
	}

	return nil
}

func platformPreDropPrivilegesInner() error {
	err := setSecurebits()
	if err != nil {
		return err
	}

	// TODO: Consider use of capability bounding sets.
	// Though should be made unnecessary by NO_NEW_PRIVS.

	return nil
}

func platformPreDropPrivileges() error {
	platformPreDropPrivilegesInner()
	// Ignore errors. SECUREBITS and modification of the capability bounding set
	// requires capabilities we may not have if we are not running as root,
	// but DropPrivileges has still been invoked because we have e.g. CAP_SYS_CHROOT.
	// The best available configuration is thus obtained, but only basic security
	// requirements (such as not running as root) are enforced.

	return nil
}

func setNoNewPrivs() error {
	err := prctl(pPR_SET_NO_NEW_PRIVS, 1, 0, 0, 0)
	if err != nil {
		return fmt.Errorf("cannot set NO_NEW_PRIVS: %v", err)
	}

	return nil
}

func platformPostDropPrivileges() error {
	return setNoNewPrivs()
}

const (
	pPR_SET_SECCOMP      = 22
	pPR_CAPBSET_DROP     = 24
	pPR_SET_SECUREBITS   = 28
	pPR_SET_NO_NEW_PRIVS = 36

	sSECBIT_NOROOT                 = 1 << 0
	sSECBIT_NOROOT_LOCKED          = 1 << 1
	sSECBIT_NO_SETUID_FIXUP        = 1 << 2
	sSECBIT_NO_SETUID_FIXUP_LOCKED = 1 << 3
	sSECBIT_KEEP_CAPS              = 1 << 4
	sSECBIT_KEEP_CAPS_LOCKED       = 1 << 5
)

func prctl(opt int, arg2, arg3, arg4, arg5 uint64) error {
	_, _, e1 := syscall.Syscall6(syscall.SYS_PRCTL, uintptr(opt),
		uintptr(arg2), uintptr(arg3), uintptr(arg4), uintptr(arg5), 0)
	if e1 != 0 {
		return e1
	}

	return nil
}
