package business

import (
	"errors"
	"testing"
)

func TestAccountType(t *testing.T) {
	t.Run("missing state defaults to regular", func(t *testing.T) {
		if got := ResolveAccountType(nil); got != AccountTypeRegular {
			t.Fatalf("ResolveAccountType(nil) = %q, want %q", got, AccountTypeRegular)
		}
	})

	t.Run("supported values parse exactly", func(t *testing.T) {
		for _, want := range []AccountType{AccountTypeRegular, AccountTypeBusiness} {
			got, err := ParseAccountType(string(want))
			if err != nil {
				t.Fatalf("ParseAccountType(%q): %v", want, err)
			}
			if got != want {
				t.Fatalf("ParseAccountType(%q) = %q, want %q", want, got, want)
			}
		}
	})

	t.Run("unsupported values are rejected", func(t *testing.T) {
		for _, raw := range []string{"", "pro", "Regular", "BUSINESS", " business"} {
			got, err := ParseAccountType(raw)
			if !errors.Is(err, ErrInvalidAccountType) {
				t.Errorf("ParseAccountType(%q) error = %v, want ErrInvalidAccountType", raw, err)
			}
			if got != "" {
				t.Errorf("ParseAccountType(%q) = %q, want zero value", raw, got)
			}
		}
	})
}
