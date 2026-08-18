package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectMigrationsDir(t *testing.T) {
	t.Parallel()

	t.Run("populated", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "000001_example.up.sql"), []byte("SELECT 1;"), 0o600); err != nil {
			t.Fatalf("write migration: %v", err)
		}

		empty, err := inspectMigrationsDir(dir)
		if err != nil {
			t.Fatalf("inspectMigrationsDir() error = %v", err)
		}
		if empty {
			t.Fatal("inspectMigrationsDir() empty = true, want false")
		}
	})

	t.Run("empty", func(t *testing.T) {
		empty, err := inspectMigrationsDir(t.TempDir())
		if err != nil {
			t.Fatalf("inspectMigrationsDir() error = %v", err)
		}
		if !empty {
			t.Fatal("inspectMigrationsDir() empty = false, want true")
		}
	})

	t.Run("missing", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "missing")
		_, err := inspectMigrationsDir(dir)
		if err == nil {
			t.Fatal("inspectMigrationsDir() error = nil, want missing-directory error")
		}
		if !strings.Contains(err.Error(), dir) {
			t.Fatalf("error %q does not name path %q", err, dir)
		}
	})

	t.Run("not a directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "migrations")
		if err := os.WriteFile(path, []byte("not a directory"), 0o600); err != nil {
			t.Fatalf("write file: %v", err)
		}
		_, err := inspectMigrationsDir(path)
		if err == nil {
			t.Fatal("inspectMigrationsDir() error = nil, want non-directory error")
		}
		if !strings.Contains(err.Error(), path) {
			t.Fatalf("error %q does not name path %q", err, path)
		}
	})
}

func TestMigrateStatusRejectsEmptyDirBeforeDatabaseConnection(t *testing.T) {
	t.Parallel()

	// The closed port proves migration-source validation happens before any
	// database connection attempt.
	_, err := runMigrateStatus("postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1", t.TempDir())
	if err == nil {
		t.Fatal("runMigrateStatus() error = nil, want empty migration bundle error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("runMigrateStatus() error = %q, want empty bundle context", err)
	}
}
