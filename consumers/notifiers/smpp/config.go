// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package smpp

import (
	"crypto/tls"
)

// Config represents SMPP transmitter configuration.
type Config struct {
	Address       string
	Username      string
	Password      string
	SystemType    string
	SourceAddrTON uint8
	SourceAddrNPI uint8
	DestAddrTON   uint8
	DestAddrNPI   uint8
	TLS           *tls.Config
}
