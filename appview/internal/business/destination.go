package business

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

var ErrInvalidDestination = errors.New("business: invalid destination")

func ValidateWebDestination(raw string) error {
	if !utf8.ValidString(raw) || len(raw) > 2048 {
		return ErrInvalidDestination
	}
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.Opaque != "" || !strings.EqualFold(parsed.Scheme, "https") || parsed.User != nil {
		return ErrInvalidDestination
	}
	if parsed.Host == "" || strings.Contains(parsed.Host, "%") || strings.ContainsAny(parsed.Host, "[]") {
		return ErrInvalidDestination
	}
	host := parsed.Hostname()
	if err := validateDNSName(host); err != nil {
		return err
	}
	if strings.Contains(parsed.Host, ":") {
		port := parsed.Port()
		if port == "" {
			return ErrInvalidDestination
		}
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return ErrInvalidDestination
		}
	}
	return nil
}

func ValidateMailtoDestination(raw string) error {
	if !strings.HasPrefix(raw, "mailto:") {
		return ErrInvalidDestination
	}
	address := strings.TrimPrefix(raw, "mailto:")
	if len(address) == 0 || len(address) > 320 || !isASCII(address) || strings.Count(address, "@") != 1 {
		return ErrInvalidDestination
	}
	if strings.ContainsAny(address, "%?#,; 	\r\n") {
		return ErrInvalidDestination
	}
	local, domain, _ := strings.Cut(address, "@")
	if !isDotAtom(local) || strings.Contains(domain, ":") {
		return ErrInvalidDestination
	}
	return validateDNSName(domain)
}

func validateDNSName(host string) error {
	if len(host) < 2 || len(host) > 253 || !isASCII(host) || strings.HasSuffix(host, ".") || net.ParseIP(host) != nil {
		return ErrInvalidDestination
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return ErrInvalidDestination
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return ErrInvalidDestination
		}
		for _, char := range label {
			if !isASCIIAlphaNumeric(char) && char != '-' {
				return ErrInvalidDestination
			}
		}
	}
	return nil
}

func isDotAtom(value string) bool {
	if value == "" || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") || strings.Contains(value, "..") {
		return false
	}
	for _, char := range value {
		if isASCIIAlphaNumeric(char) || strings.ContainsRune("!#$%&'*+-/=?^_`{|}~.", char) {
			continue
		}
		return false
	}
	return true
}

func isASCII(value string) bool {
	for _, char := range value {
		if char > 0x7f {
			return false
		}
	}
	return true
}

func isASCIIAlphaNumeric(char rune) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9'
}
