package testdb

import (
	"strings"
	"testing"
)

func TestResolveDatabaseURL(t *testing.T) {
	t.Run("test URL wins", func(t *testing.T) {
		t.Setenv("TEST_DATABASE_URL", "postgres://test")
		t.Setenv("DATABASE_URL", "postgres://fallback")
		t.Setenv("TEST_DATABASE_REQUIRED", "true")

		url, skip, err := resolveDatabaseURL()
		if err != nil {
			t.Fatalf("resolveDatabaseURL() error = %v", err)
		}
		if skip || url != "postgres://test" {
			t.Fatalf("resolveDatabaseURL() = (%q, %t), want (%q, false)", url, skip, "postgres://test")
		}
	})

	t.Run("database fallback", func(t *testing.T) {
		t.Setenv("TEST_DATABASE_URL", "")
		t.Setenv("DATABASE_URL", "postgres://fallback")
		t.Setenv("TEST_DATABASE_REQUIRED", "false")

		url, skip, err := resolveDatabaseURL()
		if err != nil {
			t.Fatalf("resolveDatabaseURL() error = %v", err)
		}
		if skip || url != "postgres://fallback" {
			t.Fatalf("resolveDatabaseURL() = (%q, %t), want (%q, false)", url, skip, "postgres://fallback")
		}
	})

	t.Run("unit-only mode may skip", func(t *testing.T) {
		t.Setenv("TEST_DATABASE_URL", "")
		t.Setenv("DATABASE_URL", "")
		t.Setenv("TEST_DATABASE_REQUIRED", "false")

		url, skip, err := resolveDatabaseURL()
		if err != nil {
			t.Fatalf("resolveDatabaseURL() error = %v", err)
		}
		if !skip || url != "" {
			t.Fatalf("resolveDatabaseURL() = (%q, %t), want (empty, true)", url, skip)
		}
	})

	t.Run("required mode fails closed", func(t *testing.T) {
		t.Setenv("TEST_DATABASE_URL", "")
		t.Setenv("DATABASE_URL", "")
		t.Setenv("TEST_DATABASE_REQUIRED", "true")

		_, _, err := resolveDatabaseURL()
		if err == nil {
			t.Fatal("resolveDatabaseURL() error = nil, want required-database error")
		}
		if !strings.Contains(err.Error(), "TEST_DATABASE_REQUIRED") {
			t.Fatalf("error %q does not name TEST_DATABASE_REQUIRED", err)
		}
	})

	t.Run("invalid required flag fails closed", func(t *testing.T) {
		t.Setenv("TEST_DATABASE_URL", "")
		t.Setenv("DATABASE_URL", "")
		t.Setenv("TEST_DATABASE_REQUIRED", "sometimes")

		_, _, err := resolveDatabaseURL()
		if err == nil {
			t.Fatal("resolveDatabaseURL() error = nil, want invalid-flag error")
		}
		if !strings.Contains(err.Error(), "TEST_DATABASE_REQUIRED") {
			t.Fatalf("error %q does not name TEST_DATABASE_REQUIRED", err)
		}
	})
}
