package port

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

func FindAvailablePort(start int) (int, error) {
	for i := start; i < start+1000; i++ {
		available, err := IsAvailable("localhost:" + strconv.Itoa(i))
		if err != nil {
			return 0, err
		} else if !available {
			continue
		}

		return i, nil
	}

	return 0, fmt.Errorf("couldn't find an available port")
}

func IsAvailable(addr string) (bool, error) {
	timeout := time.Millisecond * 500
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		// Try to create a server with the port
		server, err := net.Listen("tcp", addr)

		// if it fails then the port is likely taken
		if err != nil {
			return false, err
		}

		// close the server
		_ = server.Close()
		return true, nil
	}
	_ = conn.Close()
	return false, nil
}
