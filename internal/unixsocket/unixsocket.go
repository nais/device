package unixsocket

import (
	"fmt"
	"net"
	"os"
)

// Set up a listener on a UNIX socket.
// The socket file permissions are set accordingly.
// If the file already exists, remove it first.
func ListenWithFileMode(addr string, perm os.FileMode) (net.Listener, error) {
	err := os.Remove(addr)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("remove stale unix socket %s: %w", addr, err)
	}
	listener, err := net.Listen("unix", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on unix socket %s: %v", addr, err)
	}
	err = os.Chmod(addr, perm)
	if err != nil {
		listener.Close()
		return nil, fmt.Errorf("set permission %v on unix socket: %w", perm, err)
	}
	return listener, err
}
