package main

import (
	"testing"
)

func TestMigrateStatusEmptyDir(t *testing.T) {
	// The test passes a URL pointing at a closed port, proving that the
	// empty-dir short-circuit runs BEFORE any DB connection attempt —
	// which is the AC #9 contract.
	out, err := runMigrateStatus("postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1", t.TempDir())
	if err != nil {
		t.Fatalf("err = %v, want nil for empty migrations dir", err)
	}
	want := "no migrations applied (migrations directory is empty)"
	if out != want {
		t.Errorf("out = %q, want %q", out, want)
	}
}
