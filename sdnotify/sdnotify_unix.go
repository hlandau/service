package sdnotify

import "errors"
import "net"
import "sync"
import "os"

var SdNotifyNoSocket = errors.New("No socket")

// sdNotifySocket
var sdNotifyMutex sync.Mutex
var sdNotifySocket *net.UnixConn
var sdNotifyInited bool

// SdNotify sends a message to the init daemon. It is common to ignore the error.
//
// Taken from coreos/go-systemd/daemon. Since that code closes the socket
// after each call it won't work in a chroot. It is customized here to keep
// the socket open.
func SdNotify(state string) error {
	sdNotifyMutex.Lock()
	defer sdNotifyMutex.Unlock()

	if !sdNotifyInited {
		sdNotifyInited = true

		socketAddr := &net.UnixAddr{
			Name: os.Getenv("NOTIFY_SOCKET"),
			Net:  "unixgram",
		}

		if socketAddr.Name == "" {
			return SdNotifyNoSocket
		}

		conn, err := net.DialUnix(socketAddr.Net, nil, socketAddr)
		if err != nil {
			return err
		}

		sdNotifySocket = conn
	}

	if sdNotifySocket == nil {
		return SdNotifyNoSocket
	}

	_, err := sdNotifySocket.Write([]byte(state))
	return err
}
