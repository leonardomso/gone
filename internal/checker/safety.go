// Package checker - safety.go contains URL and address validation used to
// prevent server-side request forgery (SSRF). Without these guards an
// attacker who controls a redirect target or who plants a URL in a scanned
// document could coerce the tool into fetching cloud metadata endpoints
// (e.g. 169.254.169.254), localhost services, or internal corporate hosts
// from the user's machine.
package checker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrBlockedAddress is returned when a URL or its resolved IP falls inside
// a range that the checker refuses to contact (loopback, link-local,
// private, CGNAT, unspecified, etc.).
var ErrBlockedAddress = errors.New("blocked address")

// ErrUnsupportedScheme is returned for URLs whose scheme is not http or https.
var ErrUnsupportedScheme = errors.New("unsupported scheme")

// blockedCIDRs lists the IP ranges considered unsafe to contact during link
// checking. The list is intentionally explicit so that adding or removing a
// range is a code change with an accompanying test.
var blockedCIDRs = compileCIDRs([]string{
	// IPv4
	"0.0.0.0/8",       // "this network"
	"10.0.0.0/8",      // RFC 1918 private
	"100.64.0.0/10",   // CGNAT (RFC 6598)
	"127.0.0.0/8",     // loopback
	"169.254.0.0/16",  // link-local (incl. cloud metadata)
	"172.16.0.0/12",   // RFC 1918 private
	"192.0.0.0/24",    // IETF assignments
	"192.0.2.0/24",    // TEST-NET-1
	"192.168.0.0/16",  // RFC 1918 private
	"198.18.0.0/15",   // benchmark
	"198.51.100.0/24", // TEST-NET-2
	"203.0.113.0/24",  // TEST-NET-3
	"224.0.0.0/4",     // multicast
	"240.0.0.0/4",     // reserved
	"255.255.255.255/32",

	// IPv6
	"::/128",        // unspecified
	"::1/128",       // loopback
	"fc00::/7",      // unique local
	"fe80::/10",     // link-local
	"ff00::/8",      // multicast
	"2001:db8::/32", // documentation
})

func compileCIDRs(cidrs []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			// Programmer error - the list above is a constant.
			panic(fmt.Sprintf("checker: invalid blocked CIDR %q: %v", c, err))
		}
		out = append(out, n)
	}
	return out
}

// isBlockedIP reports whether the given IP falls in any of blockedCIDRs.
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	for _, n := range blockedCIDRs {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// validateURL parses rawURL and ensures it uses an allowed scheme and that,
// if the host is a literal IP, the IP is not in a blocked range. Hostnames
// that resolve to blocked IPs are caught later by safeDialContext.
func validateURL(rawURL string, allowPrivate bool) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("%w: %q", ErrUnsupportedScheme, u.Scheme)
	}
	if allowPrivate {
		return nil
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("%w: empty host", ErrBlockedAddress)
	}
	if ip := net.ParseIP(host); ip != nil && isBlockedIP(ip) {
		return fmt.Errorf("%w: %s", ErrBlockedAddress, ip)
	}
	return nil
}

// safeDialContext wraps a base dial function so every resolved IP is
// checked before the connection is established. It catches:
//   - hostnames that resolve to blocked IPs
//   - DNS rebinding (a host that resolves to a public IP at validate time
//     and a blocked IP at dial time)
//
// If allowPrivate is true the wrapper is a transparent pass-through.
func safeDialContext(
	base func(ctx context.Context, network, addr string) (net.Conn, error),
	allowPrivate bool,
) func(ctx context.Context, network, addr string) (net.Conn, error) {
	if allowPrivate {
		return base
	}
	resolver := net.DefaultResolver
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("split host:port: %w", err)
		}
		// Literal IP fast path.
		if ip := net.ParseIP(host); ip != nil {
			if isBlockedIP(ip) {
				return nil, fmt.Errorf("%w: %s", ErrBlockedAddress, ip)
			}
			return base(ctx, network, addr)
		}
		// Resolve and verify every returned address.
		ips, err := resolver.LookupIP(ctx, ipNetwork(network), host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if isBlockedIP(ip) {
				return nil, fmt.Errorf("%w: %s -> %s", ErrBlockedAddress, host, ip)
			}
		}
		// Re-dial using the verified address. We pick the first allowed IP
		// to avoid the resolver returning a different (possibly blocked)
		// address between LookupIP and Dial.
		return base(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
}

func ipNetwork(network string) string {
	switch network {
	case "tcp4", "udp4", "ip4":
		return "ip4"
	case "tcp6", "udp6", "ip6":
		return "ip6"
	default:
		return "ip"
	}
}
