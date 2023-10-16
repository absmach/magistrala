package options

import (
	"time"

	dtlsServer "github.com/plgd-dev/go-coap/v3/dtls/server"
	udpClient "github.com/plgd-dev/go-coap/v3/udp/client"
	udpServer "github.com/plgd-dev/go-coap/v3/udp/server"
)

// TransmissionOpt transmission options.
type TransmissionOpt struct {
	transmissionNStart             uint32
	transmissionAcknowledgeTimeout time.Duration
	transmissionMaxRetransmit      uint32
}

func (o TransmissionOpt) UDPServerApply(cfg *udpServer.Config) {
	cfg.TransmissionNStart = o.transmissionNStart
	cfg.TransmissionAcknowledgeTimeout = o.transmissionAcknowledgeTimeout
	cfg.TransmissionMaxRetransmit = o.transmissionMaxRetransmit
}

func (o TransmissionOpt) DTLSServerApply(cfg *dtlsServer.Config) {
	cfg.TransmissionNStart = o.transmissionNStart
	cfg.TransmissionAcknowledgeTimeout = o.transmissionAcknowledgeTimeout
	cfg.TransmissionMaxRetransmit = o.transmissionMaxRetransmit
}

func (o TransmissionOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.TransmissionNStart = o.transmissionNStart
	cfg.TransmissionAcknowledgeTimeout = o.transmissionAcknowledgeTimeout
	cfg.TransmissionMaxRetransmit = o.transmissionMaxRetransmit
}

// WithTransmission set options for (re)transmission for Confirmable message-s.
func WithTransmission(transmissionNStart uint32,
	transmissionAcknowledgeTimeout time.Duration,
	transmissionMaxRetransmit uint32,
) TransmissionOpt {
	return TransmissionOpt{
		transmissionNStart:             transmissionNStart,
		transmissionAcknowledgeTimeout: transmissionAcknowledgeTimeout,
		transmissionMaxRetransmit:      transmissionMaxRetransmit,
	}
}

// MTUOpt transmission options.
type MTUOpt struct {
	mtu uint16
}

func (o MTUOpt) UDPServerApply(cfg *udpServer.Config) {
	cfg.MTU = o.mtu
}

func (o MTUOpt) DTLSServerApply(cfg *dtlsServer.Config) {
	cfg.MTU = o.mtu
}

func (o MTUOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.MTU = o.mtu
}

// Setup MTU unit
func WithMTU(mtu uint16) MTUOpt {
	return MTUOpt{
		mtu: mtu,
	}
}
