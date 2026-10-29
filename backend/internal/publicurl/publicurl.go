// Package publicurl validates the PocketBase Application URL used in links
// and browser security decisions. It deliberately performs no network access.
package publicurl

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
)

var ErrInvalid = errors.New("application URL must be an absolute HTTPS origin without credentials, query, fragment, or subpath")

// Parse accepts an HTTPS origin and normalizes whitespace, scheme casing, and
// a root trailing slash. Subpaths remain reserved for a separate proxy feature.
func Parse(raw string) (*url.URL, error) {
	value := strings.TrimSpace(raw)
	if value == "" || strings.ContainsAny(value, "?#") {
		return nil, ErrInvalid
	}

	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() || parsed.Opaque != "" {
		return nil, ErrInvalid
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" {
		return nil, ErrInvalid
	}
	if parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return nil, ErrInvalid
	}
	if strings.HasSuffix(parsed.Host, ":") {
		return nil, ErrInvalid
	}
	if parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.RawFragment != "" {
		return nil, ErrInvalid
	}
	if (parsed.Path != "" && parsed.Path != "/") || (parsed.RawPath != "" && parsed.RawPath != "/") {
		return nil, ErrInvalid
	}

	hostname := parsed.Hostname()
	if net.ParseIP(hostname) == nil && !validDomain(hostname) {
		return nil, ErrInvalid
	}
	if port := parsed.Port(); port != "" {
		number, parseErr := strconv.Atoi(port)
		if parseErr != nil || number < 1 || number > 65535 {
			return nil, ErrInvalid
		}
	}

	parsed.Scheme = scheme
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawFragment = ""
	return parsed, nil
}

func validDomain(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || !isAlphaNumeric(label[0]) || !isAlphaNumeric(label[len(label)-1]) {
			return false
		}
		for i := 1; i < len(label)-1; i++ {
			if !isAlphaNumeric(label[i]) && label[i] != '-' {
				return false
			}
		}
	}
	return true
}

func isAlphaNumeric(value byte) bool {
	return (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9')
}
