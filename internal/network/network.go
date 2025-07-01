package network

import (
	"fmt"
	"net"
	"net/netip"
)

// GetOutboundIP gets the preferred outbound ip address of this machine
func GetOutboundIP() (netip.Addr, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return netip.Addr{}, fmt.Errorf("dialing to get outbound ip address: %v", err)
	}
	defer conn.Close()
	ip := conn.LocalAddr().(*net.UDPAddr).IP
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return netip.Addr{}, fmt.Errorf("parsing addr: %v", err)
	}
	return addr, nil
}
