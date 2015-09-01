package caps

import "syscall"

/*
#cgo LDFLAGS: -lcap
#include <sys/capability.h>
#include <errno.h>

static int
hasanycap(void) {
  cap_t c, zc;
  int hasCap;

  zc = cap_init();
  if (!zc) {
    return -errno;
  }

  if (cap_clear(zc) < 0) {
    cap_free(zc);
    return -errno;
  }

  c = cap_get_proc();
  if (!c) {
    cap_free(c);
    cap_free(zc);
    return -errno;
  }

  hasCap = !!cap_compare(c, zc);

  cap_free(c);
  cap_free(zc);
  return hasCap;
}
#include <stdio.h>

static int
dropcaps(void) {
  int ec;
  cap_t c;

  c = cap_init();
  if (!c) {
    cap_free(c);
    return errno;
  }

  if (cap_clear(c) < 0) {
    cap_free(c);
    return errno;
  }

  if (cap_set_proc(c)) {
    cap_free(c);
    return errno;
  }

  cap_free(c);
  return 0;
}

*/
import "C"
import "fmt"

const platformSupportsCaps = true

func ensureNone() error {
	if C.hasanycap() != 0 {
		return fmt.Errorf("still have caps")
	}

	return nil
}

func drop() error {
	eno := C.dropcaps()
	if eno != 0 {
		return syscall.Errno(eno)
	}

	return nil
}
