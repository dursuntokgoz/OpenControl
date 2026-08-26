//go:build linux

package agent

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

// describePeer returns "uid=<u> gid=<g> pid=<p>" for unix socket peers using
// SO_PEERCRED, or an error description when credentials are unavailable.
func describePeer(conn net.Conn) string {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return "non-unix"
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return "control-error"
	}
	var desc string
	var credErr error
	if cerr := raw.Control(func(fd uintptr) {
		cred, e := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if e != nil {
			credErr = e
			return
		}
		desc = fmt.Sprintf("uid=%d gid=%d pid=%d", cred.Uid, cred.Gid, cred.Pid)
	}); cerr != nil {
		return "control-error"
	}
	if credErr != nil {
		return "peercred-error"
	}
	return desc
}
