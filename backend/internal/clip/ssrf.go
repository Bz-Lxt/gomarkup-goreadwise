package clip

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"goreadwise/internal/httpx"
)

func ParseHTTPURL(raw string) (*url.URL, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("%w: url is required", httpx.ErrValidation)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid url", httpx.ErrValidation)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w: only http/https allowed", httpx.ErrDenied)
	}
	if u.Hostname() == "" {
		return nil, fmt.Errorf("%w: missing host", httpx.ErrValidation)
	}
	return u, nil
}

func ValidatePublicURL(raw string) (*url.URL, error) {
	u, err := ParseHTTPURL(raw)
	if err != nil {
		return nil, err
	}
	host := u.Hostname()
	if isForbiddenHost(host) {
		return nil, fmt.Errorf("%w: host is not allowed", httpx.ErrDenied)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		// DNS failure is not automatically trusted; still block obvious local names.
		if looksLocalName(host) {
			return nil, fmt.Errorf("%w: host is not allowed", httpx.ErrDenied)
		}
		return u, nil
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("%w: host has no addresses", httpx.ErrDenied)
	}
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return nil, fmt.Errorf("%w: resolved to private address", httpx.ErrDenied)
		}
	}
	return u, nil
}

func isForbiddenHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	switch h {
	case "localhost", "localhost.localdomain", "metadata.google.internal":
		return true
	}
	if looksLocalName(h) {
		return true
	}
	if ip := net.ParseIP(h); ip != nil && isPrivateIP(ip) {
		return true
	}
	return false
}

func looksLocalName(host string) bool {
	h := strings.ToLower(host)
	return strings.HasSuffix(h, ".local") || strings.HasSuffix(h, ".internal") || strings.HasSuffix(h, ".localhost")
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		if v4[0] == 169 && v4[1] == 254 {
			return true
		}
		// CGNAT 100.64.0.0/10
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
			return true
		}
		// benchmark / documentation
		if v4[0] == 198 && (v4[1] == 18 || v4[1] == 19) {
			return true
		}
		if v4[0] == 0 {
			return true
		}
	}
	if ip.To4() == nil {
		// unique local fc00::/7
		if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
			return true
		}
	}
	return false
}

func NormalizeHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	host = strings.TrimSuffix(host, ".")
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}
