// +build !windows

package setuid

// Setuid calls the *NIX setuid() function.
func Setuid(uid int) error {
	return setuid(uid)
}

// Setgid calls the *NIX setgid() function.
func Setgid(gid int) error {
	return setgid(gid)
}

// Setgroups calls the *NIX setgroups() function.
func Setgroups(gids []int) error {
	return setgroups(gids)
}

// Setresgid calls the *NIX setresgid() function.
func Setresgid(rgid, egid, sgid int) error {
	return setresgid(rgid, egid, sgid)
}

// Setresuid calls the *NIX setresuid() function.
func Setresuid(ruid, euid, suid int) error {
	return setresuid(ruid, euid, suid)
}
