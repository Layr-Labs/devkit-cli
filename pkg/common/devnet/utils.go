package devnet

import (
	"fmt"
	"net"
	"time"
)

// IsPortAvailable checks if a TCP port is not already bound by another service.
func IsPortAvailable(port int) bool {
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		// If dialing fails, port is likely available
		return true
	}
	_ = conn.Close()
	return false
}
