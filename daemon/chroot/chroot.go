package chroot

// Returns the "chroot anchor", the path under which the process has been
// chrooted. If the process has not been chrooted, this is "/". This can be
// used to obtain chroot-relative paths necessary to access files after
// chrooting. See ChrootRel.
func Anchor() string {
	return getAnchor()
}

// path should be an absolute path. If given the current ChrootAnchor, it can
// be accessed, returns the path which should be used to open the file given
// the current chroot and returns true. Otherwise returns false.
func Rel(path string) (chrootRelativePath string, canAddress bool) {
	return rel(path)
}
