package linkpreview

import (
	"errors"
	"net/netip"
	"slices"
	"testing"
)

// UT-010: complete DNS answer sets are accepted only when every address is
// public-routable, including after unmapping IPv4-mapped IPv6.
func TestValidateAddresses(t *testing.T) {
	t.Parallel()

	public := []netip.Addr{
		netip.MustParseAddr("8.8.8.8"),
		netip.MustParseAddr("2606:4700:4700::1111"),
	}
	got, err := ValidateAddresses(public)
	if err != nil {
		t.Fatalf("ValidateAddresses(public): %v", err)
	}
	if !slices.Equal(got, public) {
		t.Fatalf("ValidateAddresses(public) = %v, want %v", got, public)
	}
	got[0] = netip.Addr{}
	if !public[0].IsValid() {
		t.Fatal("ValidateAddresses returned the caller's mutable slice")
	}

	for _, raw := range []string{
		"0.0.0.0", "10.0.0.1", "100.64.0.1", "127.0.0.1",
		"169.254.1.1", "172.16.0.1", "192.0.0.1", "192.0.2.1",
		"192.168.1.1", "198.18.0.1", "198.51.100.1", "203.0.113.1",
		"224.0.0.1", "240.0.0.1", "255.255.255.255",
		"::", "::1", "::ffff:127.0.0.1", "::ffff:10.0.0.1",
		"::ffff:100.64.0.1", "::ffff:192.0.2.1",
		"64:ff9b::1", "64:ff9b:1::1", "100::1", "2001::1",
		"2001:2::1", "2001:10::1", "2001:20::1", "2001:db8::1",
		"2002::1", "3fff::1", "5f00::1", "fc00::1", "fe80::1",
		"fec0::1", "feff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", "ff00::1",
	} {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			_, err := ValidateAddresses([]netip.Addr{netip.MustParseAddr(raw)})
			if !errors.Is(err, ErrNotAllowed) {
				t.Fatalf("ValidateAddresses(%s) error = %v, want ErrNotAllowed", raw, err)
			}
		})
	}

	for name, addresses := range map[string][]netip.Addr{
		"empty":   nil,
		"invalid": {{}},
		"mixed IPv4": {
			netip.MustParseAddr("8.8.8.8"),
			netip.MustParseAddr("127.0.0.1"),
		},
		"mixed IPv6": {
			netip.MustParseAddr("2606:4700:4700::1111"),
			netip.MustParseAddr("fe80::1"),
		},
	} {
		name, addresses := name, addresses
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := ValidateAddresses(addresses); !errors.Is(err, ErrNotAllowed) {
				t.Fatalf("ValidateAddresses(%v) error = %v, want ErrNotAllowed", addresses, err)
			}
		})
	}
}

func TestValidateAddressesRejectsDeprecatedIPv6SiteLocal(t *testing.T) {
	t.Parallel()
	_, err := ValidateAddresses([]netip.Addr{netip.MustParseAddr("fec0::1")})
	if !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("ValidateAddresses(fec0::1) error = %v, want ErrNotAllowed", err)
	}
}

func TestValidateAddressesRejectsIPv6OutsideSupportedPublicAllocation(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"100:0:0:1::1",
		"1fff:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
		"4000::1",
		"8000::1",
		"c000::1",
	} {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			_, err := ValidateAddresses([]netip.Addr{netip.MustParseAddr(raw)})
			if !errors.Is(err, ErrNotAllowed) {
				t.Fatalf("ValidateAddresses(%s) error = %v, want ErrNotAllowed", raw, err)
			}
		})
	}
}
