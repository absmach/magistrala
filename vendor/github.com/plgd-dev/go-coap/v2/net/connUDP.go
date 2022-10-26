package net

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"go.uber.org/atomic"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// UDPConn is a udp connection provides Read/Write with context.
//
// Multiple goroutines may invoke methods on a UDPConn simultaneously.
type UDPConn struct {
	packetConn packetConn
	network    string
	connection *net.UDPConn
	errors     func(err error)
	closed     atomic.Bool
}

type ControlMessage struct {
	Src     net.IP // source address, specifying only
	IfIndex int    // interface index, must be 1 <= value when specifying
}

type packetConn interface {
	SetWriteDeadline(t time.Time) error
	WriteTo(b []byte, cm *ControlMessage, dst net.Addr) (n int, err error)
	SetMulticastInterface(ifi *net.Interface) error
	SetMulticastHopLimit(hoplim int) error
	SetMulticastLoopback(on bool) error
	JoinGroup(ifi *net.Interface, group net.Addr) error
	LeaveGroup(ifi *net.Interface, group net.Addr) error
}

type packetConnIPv4 struct {
	packetConnIPv4 *ipv4.PacketConn
}

func newPacketConnIPv4(p *ipv4.PacketConn) *packetConnIPv4 {
	return &packetConnIPv4{p}
}

func (p *packetConnIPv4) SetMulticastInterface(ifi *net.Interface) error {
	return p.packetConnIPv4.SetMulticastInterface(ifi)
}

func (p *packetConnIPv4) SetWriteDeadline(t time.Time) error {
	return p.packetConnIPv4.SetWriteDeadline(t)
}

func (p *packetConnIPv4) WriteTo(b []byte, cm *ControlMessage, dst net.Addr) (n int, err error) {
	var c *ipv4.ControlMessage
	if cm != nil {
		c = &ipv4.ControlMessage{
			Src:     cm.Src,
			IfIndex: cm.IfIndex,
		}
	}
	return p.packetConnIPv4.WriteTo(b, c, dst)
}

func (p *packetConnIPv4) SetMulticastHopLimit(hoplim int) error {
	return p.packetConnIPv4.SetMulticastTTL(hoplim)
}

func (p *packetConnIPv4) SetMulticastLoopback(on bool) error {
	return p.packetConnIPv4.SetMulticastLoopback(on)
}

func (p *packetConnIPv4) JoinGroup(ifi *net.Interface, group net.Addr) error {
	return p.packetConnIPv4.JoinGroup(ifi, group)
}

func (p *packetConnIPv4) LeaveGroup(ifi *net.Interface, group net.Addr) error {
	return p.packetConnIPv4.LeaveGroup(ifi, group)
}

type packetConnIPv6 struct {
	packetConnIPv6 *ipv6.PacketConn
}

func newPacketConnIPv6(p *ipv6.PacketConn) *packetConnIPv6 {
	return &packetConnIPv6{p}
}

func (p *packetConnIPv6) SetMulticastInterface(ifi *net.Interface) error {
	return p.packetConnIPv6.SetMulticastInterface(ifi)
}

func (p *packetConnIPv6) SetWriteDeadline(t time.Time) error {
	return p.packetConnIPv6.SetWriteDeadline(t)
}

func (p *packetConnIPv6) WriteTo(b []byte, cm *ControlMessage, dst net.Addr) (n int, err error) {
	var c *ipv6.ControlMessage
	if cm != nil {
		c = &ipv6.ControlMessage{
			Src:     cm.Src,
			IfIndex: cm.IfIndex,
		}
	}
	return p.packetConnIPv6.WriteTo(b, c, dst)
}

func (p *packetConnIPv6) SetMulticastHopLimit(hoplim int) error {
	return p.packetConnIPv6.SetMulticastHopLimit(hoplim)
}

func (p *packetConnIPv6) SetMulticastLoopback(on bool) error {
	return p.packetConnIPv6.SetMulticastLoopback(on)
}

func (p *packetConnIPv6) JoinGroup(ifi *net.Interface, group net.Addr) error {
	return p.packetConnIPv6.JoinGroup(ifi, group)
}

func (p *packetConnIPv6) LeaveGroup(ifi *net.Interface, group net.Addr) error {
	return p.packetConnIPv6.LeaveGroup(ifi, group)
}

func (p *packetConnIPv6) SetControlMessage(on bool) error {
	return p.packetConnIPv6.SetMulticastLoopback(on)
}

// IsIPv6 return's true if addr is IPV6.
func IsIPv6(addr net.IP) bool {
	if ip := addr.To16(); ip != nil && ip.To4() == nil {
		return true
	}
	return false
}

var defaultUDPConnOptions = udpConnOptions{
	errors: func(err error) {
		// don't log any error from fails for multicast requests
	},
}

type udpConnOptions struct {
	errors func(err error)
}

func NewListenUDP(network, addr string, opts ...UDPOption) (*UDPConn, error) {
	listenAddress, err := net.ResolveUDPAddr(network, addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP(network, listenAddress)
	if err != nil {
		return nil, err
	}
	return NewUDPConn(network, conn, opts...), nil
}

// NewUDPConn creates connection over net.UDPConn.
func NewUDPConn(network string, c *net.UDPConn, opts ...UDPOption) *UDPConn {
	cfg := defaultUDPConnOptions
	for _, o := range opts {
		o.applyUDP(&cfg)
	}

	var pc packetConn
	if IsIPv6(c.LocalAddr().(*net.UDPAddr).IP) {
		pc = newPacketConnIPv6(ipv6.NewPacketConn(c))
	} else {
		pc = newPacketConnIPv4(ipv4.NewPacketConn(c))
	}

	return &UDPConn{
		network:    network,
		connection: c,
		packetConn: pc,
		errors:     cfg.errors,
	}
}

// LocalAddr returns the local network address. The Addr returned is shared by all invocations of LocalAddr, so do not modify it.
func (c *UDPConn) LocalAddr() net.Addr {
	return c.connection.LocalAddr()
}

// RemoteAddr returns the remote network address. The Addr returned is shared by all invocations of RemoteAddr, so do not modify it.
func (c *UDPConn) RemoteAddr() net.Addr {
	return c.connection.RemoteAddr()
}

// Network name of the network (for example, udp4, udp6, udp)
func (c *UDPConn) Network() string {
	return c.network
}

// Close closes the connection.
func (c *UDPConn) Close() error {
	if !c.closed.CAS(false, true) {
		return nil
	}
	return c.connection.Close()
}

func (c *UDPConn) writeToAddr(iface *net.Interface, src *net.IP, multicastHopLimit int, raddr *net.UDPAddr, buffer []byte) error {
	var pktSrc net.IP
	var p packetConn
	if IsIPv6(raddr.IP) {
		p = newPacketConnIPv6(ipv6.NewPacketConn(c.connection))
		pktSrc = net.IPv6zero
	} else {
		p = newPacketConnIPv4(ipv4.NewPacketConn(c.connection))
		pktSrc = net.IPv4zero
	}
	if src != nil {
		pktSrc = *src
	}

	if c.closed.Load() {
		return ErrConnectionIsClosed
	}
	if iface != nil {
		if err := p.SetMulticastInterface(iface); err != nil {
			return err
		}
	}
	if err := p.SetMulticastHopLimit(multicastHopLimit); err != nil {
		return err
	}

	var err error
	if iface != nil || src != nil {
		_, err = p.WriteTo(buffer, &ControlMessage{
			Src:     pktSrc,
			IfIndex: iface.Index,
		}, raddr)
	} else {
		_, err = p.WriteTo(buffer, nil, raddr)
	}
	return err
}

func filterAddressesByNetwork(network string, ifaceAddrs []net.Addr) []net.Addr {
	filtered := make([]net.Addr, 0, len(ifaceAddrs))
	for _, srcAddr := range ifaceAddrs {
		addrMask := srcAddr.String()
		addr := strings.Split(addrMask, "/")[0]
		if strings.Contains(addr, ":") && network == "udp4" {
			continue
		}
		if !strings.Contains(addr, ":") && network == "udp6" {
			continue
		}
		filtered = append(filtered, srcAddr)
	}
	return filtered
}

func convAddrsToIps(ifaceAddrs []net.Addr) []net.IP {
	ips := make([]net.IP, 0, len(ifaceAddrs))
	for _, addr := range ifaceAddrs {
		addrMask := addr.String()
		addr := strings.Split(addrMask, "/")[0]
		ip := net.ParseIP(addr)
		if ip != nil {
			ips = append(ips, ip)
		}
	}
	return ips
}

// WriteMulticast sends multicast to the remote multicast address.
// By default it is sent over all network interfaces and all compatible source IP addresses with hop limit 1.
// Via opts you can specify the network interface, source IP address, and hop limit.
func (c *UDPConn) WriteMulticast(ctx context.Context, raddr *net.UDPAddr, buffer []byte, opts ...MulticastOption) error {
	opt := MulticastOptions{
		HopLimit: 1,
	}
	for _, o := range opts {
		o.applyMC(&opt)
	}
	return c.writeMulticast(ctx, raddr, buffer, opt)
}

func (c *UDPConn) writeMulticastWithInterface(ctx context.Context, raddr *net.UDPAddr, buffer []byte, opt MulticastOptions) error {
	if opt.Iface == nil && opt.IFaceMode == MulticastSpecificInterface {
		return fmt.Errorf("invalid interface")
	}
	if opt.Source != nil {
		return c.writeToAddr(opt.Iface, opt.Source, opt.HopLimit, raddr, buffer)
	}
	ifaceAddrs, err := opt.Iface.Addrs()
	if err != nil {
		return err
	}
	netType := "udp4"
	if IsIPv6(raddr.IP) {
		netType = "udp6"
	}
	var errors []error
	for _, ip := range convAddrsToIps(filterAddressesByNetwork(netType, ifaceAddrs)) {
		ipAddr := ip
		opt.Source = &ipAddr
		err = c.writeToAddr(opt.Iface, opt.Source, opt.HopLimit, raddr, buffer)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if errors == nil {
		return nil
	}
	if len(errors) == 1 {
		return errors[0]
	}
	return fmt.Errorf("%v", errors)
}

func (c *UDPConn) writeMulticastToAllInterfaces(ctx context.Context, raddr *net.UDPAddr, buffer []byte, opt MulticastOptions) error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("cannot get interfaces for multicast connection: %w", err)
	}

	var errors []error
	for _, iface := range ifaces {
		if iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		if iface.Flags&net.FlagUp != net.FlagUp {
			continue
		}
		specificOpt := opt
		specificOpt.Iface = &iface
		specificOpt.IFaceMode = MulticastSpecificInterface
		err = c.writeMulticastWithInterface(ctx, raddr, buffer, specificOpt)
		if err != nil {
			if opt.InterfaceError != nil {
				opt.InterfaceError(&iface, err)
				continue
			}
			errors = append(errors, err)
		}
	}
	if errors == nil {
		return nil
	}
	if len(errors) == 1 {
		return errors[0]
	}
	return fmt.Errorf("%v", errors)
}

func (c *UDPConn) validateMulticast(ctx context.Context, raddr *net.UDPAddr, buffer []byte, opt MulticastOptions) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if raddr == nil {
		return fmt.Errorf("cannot write multicast with context: invalid raddr")
	}
	if _, ok := c.packetConn.(*packetConnIPv4); ok && IsIPv6(raddr.IP) {
		return fmt.Errorf("cannot write multicast with context: invalid destination address(%v)", raddr.IP)
	}
	if opt.Source != nil && IsIPv6(*opt.Source) && !IsIPv6(raddr.IP) {
		return fmt.Errorf("cannot write multicast with context: invalid source address(%v) for destination(%v)", opt.Source, raddr.IP)
	}
	return nil
}

func (c *UDPConn) writeMulticast(ctx context.Context, raddr *net.UDPAddr, buffer []byte, opt MulticastOptions) error {
	err := c.validateMulticast(ctx, raddr, buffer, opt)
	if err != nil {
		return err
	}

	switch opt.IFaceMode {
	case MulticastAllInterface:
		err := c.writeMulticastToAllInterfaces(ctx, raddr, buffer, opt)
		if err != nil {
			return fmt.Errorf("cannot write multicast to all interfaces: %w", err)
		}
	case MulticastAnyInterface:
		err := c.writeToAddr(nil, opt.Source, opt.HopLimit, raddr, buffer)
		if err != nil {
			return fmt.Errorf("cannot write multicast to any: %w", err)
		}
	case MulticastSpecificInterface:
		err := c.writeMulticastWithInterface(ctx, raddr, buffer, opt)
		if err != nil {
			if opt.InterfaceError != nil {
				opt.InterfaceError(opt.Iface, err)
				return nil
			}
			return fmt.Errorf("cannot write multicast to %v: %w", opt.Iface.Name, err)
		}
	}
	return nil
}

// WriteWithContext writes data with context.
func (c *UDPConn) WriteWithContext(ctx context.Context, raddr *net.UDPAddr, buffer []byte) error {
	if raddr == nil {
		return fmt.Errorf("cannot write with context: invalid raddr")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if c.closed.Load() {
		return ErrConnectionIsClosed
	}
	n, err := WriteToUDP(c.connection, raddr, buffer)
	if err != nil {
		return err
	}
	if n != len(buffer) {
		return ErrWriteInterrupted
	}

	return nil
}

// ReadWithContext reads packet with context.
func (c *UDPConn) ReadWithContext(ctx context.Context, buffer []byte) (int, *net.UDPAddr, error) {
	select {
	case <-ctx.Done():
		return -1, nil, ctx.Err()
	default:
	}
	if c.closed.Load() {
		return -1, nil, ErrConnectionIsClosed
	}
	n, s, err := c.connection.ReadFromUDP(buffer)
	if err != nil {
		return -1, nil, fmt.Errorf("cannot read from udp connection: %w", err)
	}
	return n, s, err
}

// SetMulticastLoopback sets whether transmitted multicast packets
// should be copied and send back to the originator.
func (c *UDPConn) SetMulticastLoopback(on bool) error {
	return c.packetConn.SetMulticastLoopback(on)
}

// JoinGroup joins the group address group on the interface ifi.
// By default all sources that can cast data to group are accepted.
// It's possible to mute and unmute data transmission from a specific
// source by using ExcludeSourceSpecificGroup and
// IncludeSourceSpecificGroup.
// JoinGroup uses the system assigned multicast interface when ifi is
// nil, although this is not recommended because the assignment
// depends on platforms and sometimes it might require routing
// configuration.
func (c *UDPConn) JoinGroup(ifi *net.Interface, group net.Addr) error {
	return c.packetConn.JoinGroup(ifi, group)
}

// LeaveGroup leaves the group address group on the interface ifi
// regardless of whether the group is any-source group or source-specific group.
func (c *UDPConn) LeaveGroup(ifi *net.Interface, group net.Addr) error {
	return c.packetConn.LeaveGroup(ifi, group)
}
