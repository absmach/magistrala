package net

import "net"

// A UDPOption sets options such as errors parameters, etc.
type UDPOption interface {
	applyUDP(*udpConnOptions)
}

type ErrorsOpt struct {
	errors func(err error)
}

func (h ErrorsOpt) applyUDP(o *udpConnOptions) {
	o.errors = h.errors
}

func WithErrors(v func(err error)) ErrorsOpt {
	return ErrorsOpt{
		errors: v,
	}
}

func DefaultMulticastOptions() MulticastOptions {
	return MulticastOptions{
		IFaceMode: MulticastAllInterface,
		HopLimit:  1,
	}
}

type MulticastInterfaceMode int

const (
	MulticastAllInterface      MulticastInterfaceMode = 0
	MulticastAnyInterface      MulticastInterfaceMode = 1
	MulticastSpecificInterface MulticastInterfaceMode = 2
)

type InterfaceError = func(iface *net.Interface, err error)

type MulticastOptions struct {
	IFaceMode      MulticastInterfaceMode
	Iface          *net.Interface
	Source         *net.IP
	HopLimit       int
	InterfaceError InterfaceError
}

func (m *MulticastOptions) Apply(o MulticastOption) {
	o.applyMC(m)
}

// A MulticastOption sets options such as hop limit, etc.
type MulticastOption interface {
	applyMC(*MulticastOptions)
}

type MulticastInterfaceModeOpt struct {
	mode MulticastInterfaceMode
}

func (m MulticastInterfaceModeOpt) applyMC(o *MulticastOptions) {
	o.IFaceMode = m.mode
}

func WithAnyMulticastInterface() MulticastOption {
	return MulticastInterfaceModeOpt{mode: MulticastAnyInterface}
}

func WithAllMulticastInterface() MulticastOption {
	return MulticastInterfaceModeOpt{mode: MulticastAllInterface}
}

type MulticastInterfaceOpt struct {
	iface net.Interface
}

func (m MulticastInterfaceOpt) applyMC(o *MulticastOptions) {
	o.Iface = &m.iface
	o.IFaceMode = MulticastSpecificInterface
}

func WithMulticastInterface(iface net.Interface) MulticastOption {
	return &MulticastInterfaceOpt{iface: iface}
}

type MulticastHoplimitOpt struct {
	hoplimit int
}

func (m MulticastHoplimitOpt) applyMC(o *MulticastOptions) {
	o.HopLimit = m.hoplimit
}
func WithMulticastHoplimit(hoplimit int) MulticastOption {
	return &MulticastHoplimitOpt{hoplimit: hoplimit}
}

type MulticastSourceOpt struct {
	source net.IP
}

func (m MulticastSourceOpt) applyMC(o *MulticastOptions) {
	o.Source = &m.source
}
func WithMulticastSource(source net.IP) MulticastOption {
	return &MulticastSourceOpt{source: source}
}

type MulticastInterfaceErrorOpt struct {
	interfaceError InterfaceError
}

func (m MulticastInterfaceErrorOpt) applyMC(o *MulticastOptions) {
	o.InterfaceError = m.interfaceError
}

// WithMulticastInterfaceError sets the callback for interface errors. If it is set error is not propagated as result of WriteMulticast.
func WithMulticastInterfaceError(interfaceError InterfaceError) MulticastOption {
	return &MulticastInterfaceErrorOpt{interfaceError: interfaceError}
}
