//go:build !linux

package agent

import "net"

// describePeer returns a stable identifier for non-Linux platforms where
// SO_PEERCRED is unavailable. Token authentication remains mandatory.
func describePeer(conn net.Conn) string {
	return conn.RemoteAddr().String()
}
