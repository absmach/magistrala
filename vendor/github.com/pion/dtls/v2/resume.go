// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package dtls

import (
	"context"
	"net"
)

// Resume imports an already established dtls connection using a specific dtls state
func Resume(state *State, conn net.PacketConn, rAddr net.Addr, config *Config) (*Conn, error) {
	if err := state.initCipherSuite(); err != nil {
		return nil, err
	}
	c, err := createConn(context.Background(), conn, rAddr, config, state.isClient, state)
	if err != nil {
		return nil, err
	}

	return c, nil
}
