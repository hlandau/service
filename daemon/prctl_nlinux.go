// +build !linux

package daemon

func platformPreDropPrivileges() error {
	return nil
}

func platformPostDropPrivileges() error {
	return nil
}
