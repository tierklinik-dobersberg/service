package utils

import (
	"net"

	"github.com/tierklinik-dobersberg/logger"
)

// IPNetworks provides some useful utility methods for operating
// on a slice of IP networks.
type IPNetworks []net.IPNet

// ParseNetworks parses a slice of IP CIDR network definitions.
func ParseNetworks(nets []string) (IPNetworks, error) {
	result := make([]net.IPNet, len(nets))

	for idx, n := range nets {
		_, ipnet, err := net.ParseCIDR(n)
		if err != nil {
			return nil, err
		}

		result[idx] = *ipnet
	}

	return IPNetworks(result), nil
}

// Contains returns true if ip is contained in at least one
// of the IP networks from nets.
func (nets IPNetworks) Contains(ip net.IP) bool {
	for _, net := range nets {
		if net.Contains(ip) {
			return true
		}
	}
	return false
}

// ContainsString is like Contains but accepts the IP address
// in it's string format. If ip is not a valid IP false is
// returned.
func (nets IPNetworks) ContainsString(ip string) bool {
	parsed := ParseIP(ip)
	if parsed == nil {
		logger.DefaultLogger().Errorf("failed to parse %s as an IP", ip)
		return false
	}

	return nets.Contains(parsed)
}

// ParseIP is like net.ParseIP but accepts that ip may be
// enclosed in [].
func ParseIP(s string) net.IP {
	if s[0] == '[' && s[len(s)-1] == ']' {
		s = s[1 : len(s)-1]
	}

	return net.ParseIP(s)
}
