package process

import (
	"net"
	"strconv"

	"github.com/pkg/errors"
)

// CreateListener creates a listener on a port
// Pass a prefered port (will increment by 1 if port is not available)
// or pass 0 to auto-find any available port
func CreateListener(preferedPort int) (net.Listener, int, error) {
	var ln net.Listener
	var err error
	port := preferedPort
	max := 50
	for {
		// we really want to test availability on 127.0.0.1
		ln, err = net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err == nil {
			ln.Close()
			// but then, we want to listen to as many local IP's as possible
			ln, err = net.Listen("tcp", ":"+strconv.Itoa(port))
			if err == nil {
				break
			}
		}
		if preferedPort == 0 {
			return nil, 0, errors.Wrap(err, "unable to find an available port")
		}
		max--
		if max == 0 {
			return nil, 0, errors.Wrapf(err, "unable to find an available port (from %d to %d)", preferedPort, port)
		}
		port++
	}
	if preferedPort == 0 {
		port = ln.Addr().(*net.TCPAddr).Port
	}
	return ln, port, nil
}
