package urlutil

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

// ErrInvalidURL is returned when a URL fails validation.
var ErrInvalidURL = errors.New("invalid URL")

// ErrInternalURL is returned when a URL points to an internal/private address.
var ErrInternalURL = errors.New("URL points to internal address")

// ValidateExternalURL validates that a URL is a valid external HTTP/HTTPS URL
// and does not point to internal/private IP ranges.
func ValidateExternalURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ErrInvalidURL
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidURL
	}

	if u.Host == "" {
		return ErrInvalidURL
	}

	host := u.Hostname()
	if host == "" {
		return ErrInvalidURL
	}

	// Block localhost and loopback
	if isLoopback(host) {
		return ErrInternalURL
	}

	// Block private IP ranges
	ip := net.ParseIP(host)
	if ip != nil {
		if isPrivateIP(ip) {
			return ErrInternalURL
		}
	}

	// For hostnames, resolve and check IPs (best effort)
	// We do a basic check for common internal hostnames
	if isInternalHostname(host) {
		return ErrInternalURL
	}

	return nil
}

// isLoopback checks if the host is a loopback address.
func isLoopback(host string) bool {
	return host == "localhost" || host == "localhost.localdomain" ||
		host == "127.0.0.1" || host == "::1" || host == "0.0.0.0"
}

// isPrivateIP checks if the IP is in a private/reserved range.
func isPrivateIP(ip net.IP) bool {
	// Loopback
	if ip.IsLoopback() {
		return true
	}
	// Private ranges
	if ip.IsPrivate() {
		return true
	}
	// Link-local (169.254.0.0/16, fe80::/10)
	if ip.IsLinkLocalUnicast() {
		return true
	}
	// Unspecified (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return true
	}
	// Multicast
	if ip.IsMulticast() {
		return true
	}
	// Documentation addresses (192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24, 2001:db8::/32)
	if isDocumentationIP(ip) {
		return true
	}
	// Cloud metadata (169.254.169.254)
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return true
	}
	return false
}

// isDocumentationIP checks for RFC 5737 / RFC 3849 documentation addresses.
func isDocumentationIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		// 192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24
		if ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 2 {
			return true
		}
		if ip4[0] == 198 && ip4[1] == 51 && ip4[2] == 100 {
			return true
		}
		if ip4[0] == 203 && ip4[1] == 0 && ip4[2] == 113 {
			return true
		}
	}
	// 2001:db8::/32
	if len(ip) == 16 && ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x0d && ip[3] == 0xb8 {
		return true
	}
	return false
}

// isInternalHostname checks for common internal hostnames.
// This is a best-effort check; DNS resolution would be more thorough.
func isInternalHostname(host string) bool {
	internalSuffixes := []string{
		".local", ".internal", ".corp", ".lan", ".home",
		".docker", ".kubernetes", ".k8s", ".consul",
	}
	lower := strings.ToLower(host)
	for _, s := range internalSuffixes {
		if strings.HasSuffix(lower, s) {
			return true
		}
	}
	// AWS metadata
	if lower == "metadata" || lower == "metadata.google.internal" {
		return true
	}
	return false
}

// ValidateProbeURLs validates a list of probe URLs against a target URL.
// All probe URLs must be on the same host as the target URL.
func ValidateProbeURLs(targetURL string, probeURLs []string) error {
	target, err := url.Parse(targetURL)
	if err != nil {
		return err
	}
	targetHost := target.Hostname()

	for _, p := range probeURLs {
		if err := ValidateExternalURL(p); err != nil {
			return err
		}
		pu, err := url.Parse(p)
		if err != nil {
			return err
		}
		if pu.Hostname() != targetHost {
			return errors.New("probe URL host must match target URL host")
		}
	}
	return nil
}