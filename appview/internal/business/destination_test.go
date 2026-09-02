package business

import (
	"errors"
	"strings"
	"testing"
)

func TestDestinationValidation(t *testing.T) {
	webPrefix := "https://example.com/"
	webAtLimit := webPrefix + strings.Repeat("a", 2048-len(webPrefix))
	validWeb := []string{
		"https://example.com",
		"HTTPS://EXAMPLE.COM:65535/path?query=yes#fragment",
		"https://xn--bcher-kva.example",
		"https://" + strings.Repeat("a", 63) + ".example",
		webAtLimit,
	}
	for _, destination := range validWeb {
		if err := ValidateWebDestination(destination); err != nil {
			t.Errorf("ValidateWebDestination(%q): %v", destination, err)
		}
	}

	invalidWeb := []string{
		"http://example.com",
		"custom://example.com",
		"https:opaque",
		"https:///path",
		"https://user:password@example.com",
		"https://localhost",
		"https://example.com.",
		"https://bücher.example",
		"https://127.0.0.1",
		"https://[2001:db8::1]",
		"https://%65xample.com",
		"https://-bad.example",
		"https://bad-.example",
		"https://" + strings.Repeat("a", 64) + ".example",
		"https://example.com:0",
		"https://example.com:65536",
		"https://example.com:notaport",
		webAtLimit + "a",
	}
	for _, destination := range invalidWeb {
		if err := ValidateWebDestination(destination); !errors.Is(err, ErrInvalidDestination) {
			t.Errorf("ValidateWebDestination(%q) error = %v, want ErrInvalidDestination", destination, err)
		}
	}

	mailAtLimit := "mailto:" + strings.Repeat("a", 308) + "@example.com"
	validMail := []string{
		"mailto:person@example.com",
		"mailto:First.Last+shop@EXAMPLE.COM",
		mailAtLimit,
	}
	for _, destination := range validMail {
		if err := ValidateMailtoDestination(destination); err != nil {
			t.Errorf("ValidateMailtoDestination(%q): %v", destination, err)
		}
	}

	invalidMail := []string{
		"MAILTO:person@example.com",
		"mailto:person@localhost",
		"mailto:person@example.com:25",
		"mailto:person@127.0.0.1",
		"mailto:person@bücher.example",
		"mailto:person@example.com.",
		"mailto:@example.com",
		"mailto:.person@example.com",
		"mailto:person.@example.com",
		"mailto:person..other@example.com",
		"mailto:person@@example.com",
		"mailto:person name@example.com",
		"mailto:person%2Bshop@example.com",
		"mailto:person@example.com?subject=hello",
		"mailto:person@example.com#fragment",
		"mailto:one@example.com,two@example.com",
		"mailto:one@example.com;two@example.com",
		"mailto:person\n@example.com",
		mailAtLimit[:len(mailAtLimit)-len("@example.com")] + "a@example.com",
	}
	for _, destination := range invalidMail {
		if err := ValidateMailtoDestination(destination); !errors.Is(err, ErrInvalidDestination) {
			t.Errorf("ValidateMailtoDestination(%q) error = %v, want ErrInvalidDestination", destination, err)
		}
	}
}
