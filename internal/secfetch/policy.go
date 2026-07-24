// Package secfetch provides an outbound HTTP client for MCP tools that fetch
// arbitrary internet URLs, hardened against SSRF: it only reaches hosts on an
// explicit allowlist, and independently refuses to connect to any IP in a
// private/loopback/link-local/multicast/cloud-metadata range even if DNS for
// an allowed hostname resolves there (rebinding).
package secfetch

import (
	"net"
	"strings"
)

// extraBlockedNets covers reserved ranges not already caught by net.IP's
// IsPrivate/IsLoopback/IsLinkLocalUnicast/IsMulticast helpers, notably the
// shared CGNAT range (100.64.0.0/10) and the IANA special-purpose blocks
// used for documentation/benchmarking, which have no legitimate fetch target.
// IPv4-mapped IPv6 addresses (::ffff:a.b.c.d) don't need a separate entry
// here: isBlockedIP unwraps them to their 4-byte form via ip.To4() before
// these checks run.
var extraBlockedNets = mustParseCIDRs([]string{
	"100.64.0.0/10",   // Shared Address Space (CGNAT)
	"192.0.0.0/24",    // IETF Protocol Assignments
	"192.0.2.0/24",    // TEST-NET-1
	"198.18.0.0/15",   // Benchmarking
	"198.51.100.0/24", // TEST-NET-2
	"203.0.113.0/24",  // TEST-NET-3
})

func mustParseCIDRs(cidrs []string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic("secfetch: invalid CIDR " + c + ": " + err.Error())
		}
		nets = append(nets, n)
	}
	return nets
}

// isBlockedIP reports whether ip must never be reached by the fetch client:
// unspecified, loopback, link-local, multicast, RFC1918/ULA private space,
// or one of the extra reserved ranges above. This includes the cloud
// metadata address 169.254.169.254, which falls under IsLinkLocalUnicast.
func isBlockedIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}
	if ip.IsUnspecified() || ip.IsLoopback() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsPrivate() {
		return true
	}
	for _, n := range extraBlockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// hostAllowed reports whether host is permitted to be fetched: it must match
// an entry in allowed and must not match any entry in denied (denied wins on
// conflict). An empty allowed list matches nothing — the policy is
// fail-closed by default, not fail-open.
func hostAllowed(host string, allowed, denied []string) bool {
	host = strings.ToLower(host)
	for _, pattern := range denied {
		if matchesHost(host, pattern) {
			return false
		}
	}
	for _, pattern := range allowed {
		if matchesHost(host, pattern) {
			return true
		}
	}
	return false
}

// matchesHost reports whether host equals pattern, or is a subdomain of
// pattern when pattern is written as "*.example.com".
func matchesHost(host, pattern string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" {
		return false
	}
	if base, ok := strings.CutPrefix(pattern, "*."); ok {
		return host == base || strings.HasSuffix(host, "."+base)
	}
	return host == pattern
}
