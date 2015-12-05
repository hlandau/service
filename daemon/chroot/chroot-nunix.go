// +build windows plan9

package chroot

func getAnchor() string {
	return ""
}

func rel(path string) (chrootRelativePath string, canAddress bool) {
	return path, true
}
