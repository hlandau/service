// +build cgo,unix

package gsptcall

import "github.com/erikdubbelboer/gspt"

func setProcTitle(title string) {
	gspt.SetProcTitle(title)
}
