// +build !windows

#define _POSIX_C_SOURCE 1
#include <sys/types.h>
#include <pwd.h>
#include <grp.h>
#include <unistd.h>
#include <stdlib.h>
#include <errno.h>

void de_gid_cb(void *ptr, gid_t gid);

int de_username_to_uid(const char *name, uid_t *uid) {
  struct passwd p, *pp = NULL;
  size_t buflen = 1024;
  char *buf = NULL;
  int ec;
  long sz;

  sz = sysconf(_SC_GETPW_R_SIZE_MAX);
  if (sz > buflen)
    buflen = sz;

again:
  buf = realloc(buf, buflen);
  if (!buf)
    return -1;
  ec = getpwnam_r(name, &p, buf, buflen, &pp);
  if (ec == ERANGE) {
    buflen *= 2;
    goto again;
  }
  if (ec != 0 || !pp) {
    free(buf);
    return -1;
  }

  *uid = p.pw_uid;
  free(buf);
  return 0;
}

int de_groupname_to_gid(const char *name, gid_t *gid) {
  struct group p, *pp = NULL;
  size_t buflen = 1024;
  char *buf = NULL;
  int ec;
  long sz;

  sz = sysconf(_SC_GETGR_R_SIZE_MAX);
  if (sz > buflen)
    buflen = sz;

again:
  buf = realloc(buf, buflen);
  if (!buf)
    return -1;
  ec = getgrnam_r(name, &p, buf, buflen, &pp);
  if (ec == ERANGE) {
    buflen *= 2;
    goto again;
  }
  if (ec != 0 || !pp) {
    free(buf);
    return -1;
  }

  *gid = p.gr_gid;
  free(buf);
  return 0;
}

int de_get_extra_gids(gid_t gid, void *p) {
  struct group g, *pg = NULL;
  size_t buflen = 1024;
  char *buf = NULL;
  int ec;
  long sz;
  char **name;
  gid_t agid;

  sz = sysconf(_SC_GETGR_R_SIZE_MAX);
  if (sz > buflen)
    buflen = sz;

again:
  buf = realloc(buf, buflen);
  if (!buf)
    return -1;
  ec = getgrgid_r(gid, &g, buf, buflen, &pg);
  if (ec == ERANGE) {
    buflen *= 2;
    goto again;
  }
  if (ec != 0 || !pg) {
    free(buf);
    return -1;
  }

  for (name=g.gr_mem; *name; ++name) {
    ec = de_groupname_to_gid(*name, &agid);
    if (ec < 0) {
      free(buf);
      return -1;
    }
    de_gid_cb(p, agid);
  }

  free(buf);
  return 0;
}

int de_gid_for_uid(uid_t uid, gid_t *gid) {
  struct passwd p, *pp = NULL;
  size_t buflen = 1024;
  char *buf = NULL;
  int ec;
  long sz;

  sz = sysconf(_SC_GETGR_R_SIZE_MAX);
  if (sz > buflen)
    buflen = sz;

again:
  buf = realloc(buf, buflen);
  if (!buf)
    return -1;
  ec = getpwuid_r(uid, &p, buf, buflen, &pp);
  if (ec == ERANGE) {
    buflen *= 2;
    goto again;
  }
  if (ec != 0 || !pp) {
    free(buf);
    return -1;
  }

  *gid = p.pw_gid;

  free(buf);
  return 0;
}
