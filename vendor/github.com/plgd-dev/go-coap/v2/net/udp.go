package net

import (
	"net"
)

// WriteToUDP acts just like net.UDPConn.WriteTo(), but uses a *SessionUDP instead of a net.Addr.
func WriteToUDP(conn *net.UDPConn, raddr *net.UDPAddr, b []byte) (int, error) {
	if conn.RemoteAddr() == nil {
		// Connection remote address must be nil otherwise
		// "WriteTo with pre-connected connection" will be thrown
		return conn.WriteToUDP(b, raddr)
	}
	return conn.Write(b)
}
