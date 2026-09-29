package testdb

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestResolveDatabaseURL(t *testing.T) {
	t.Run("test URL wins", func(t *testing.T) {
		t.Setenv("TEST_DATABASE_URL", "postgres://user:secret@localhost/craftsky_test")
		t.Setenv("DATABASE_URL", "postgres://user:secret@localhost/craftsky_dev")
		t.Setenv("TEST_DATABASE_ALLOW_DATABASE_URL", "true")
		t.Setenv("TEST_DATABASE_REQUIRED", "true")

		url, skip, err := resolveDatabaseURL()
		if err != nil {
			t.Fatalf("resolveDatabaseURL() error = %v", err)
		}
		if skip || url != "postgres://user:secret@localhost/craftsky_test" {
			t.Fatalf("resolveDatabaseURL() = (%q, %t), want test URL", url, skip)
		}
	})

	t.Run("database fallback requires opt in", func(t *testing.T) {
		t.Setenv("TEST_DATABASE_URL", "")
		t.Setenv("DATABASE_URL", "postgres://user:secret@localhost/craftsky_dev")
		t.Setenv("TEST_DATABASE_ALLOW_DATABASE_URL", "")
		t.Setenv("TEST_DATABASE_REQUIRED", "false")

		url, skip, err := resolveDatabaseURL()
		if err != nil {
			t.Fatalf("resolveDatabaseURL() error = %v", err)
		}
		if !skip || url != "" {
			t.Fatalf("resolveDatabaseURL() = (%q, %t), want (empty, true)", url, skip)
		}
	})

	t.Run("opted-in safe database fallback", func(t *testing.T) {
		t.Setenv("TEST_DATABASE_URL", "")
		t.Setenv("DATABASE_URL", "postgres://user:secret@localhost/craftsky_dev")
		t.Setenv("TEST_DATABASE_ALLOW_DATABASE_URL", "true")

		url, skip, err := resolveDatabaseURL()
		if err != nil {
			t.Fatalf("resolveDatabaseURL() error = %v", err)
		}
		if skip || url != "postgres://user:secret@localhost/craftsky_dev" {
			t.Fatalf("resolveDatabaseURL() = (%q, %t), want opted-in fallback", url, skip)
		}
	})

	t.Run("unsafe test URL is rejected", func(t *testing.T) {
		t.Setenv("TEST_DATABASE_URL", "postgres://user:secret@db.example/craftsky")
		t.Setenv("TEST_DATABASE_REQUIRED", "true")

		_, _, err := resolveDatabaseURL()
		if err == nil || !strings.Contains(err.Error(), "refusing database") {
			t.Fatalf("resolveDatabaseURL() error = %v, want unsafe-target error", err)
		}
		if strings.Contains(err.Error(), "secret") {
			t.Fatalf("resolveDatabaseURL() leaked credentials: %v", err)
		}
	})

	t.Run("unsafe opted-in fallback is rejected", func(t *testing.T) {
		t.Setenv("TEST_DATABASE_URL", "")
		t.Setenv("DATABASE_URL", "postgres://user:secret@db.example/craftsky")
		t.Setenv("TEST_DATABASE_ALLOW_DATABASE_URL", "true")

		_, _, err := resolveDatabaseURL()
		if err == nil || !strings.Contains(err.Error(), "refusing database") {
			t.Fatalf("resolveDatabaseURL() error = %v, want unsafe-target error", err)
		}
	})

	t.Run("marker must be a database name token", func(t *testing.T) {
		t.Setenv("TEST_DATABASE_URL", "postgres://user@localhost/contest_production")

		_, _, err := resolveDatabaseURL()
		if err == nil || !strings.Contains(err.Error(), "refusing database") {
			t.Fatalf("resolveDatabaseURL() error = %v, want token-boundary error", err)
		}
	})

	t.Run("required mode fails closed", func(t *testing.T) {
		t.Setenv("TEST_DATABASE_URL", "")
		t.Setenv("DATABASE_URL", "postgres://user:secret@localhost/craftsky_dev")
		t.Setenv("TEST_DATABASE_ALLOW_DATABASE_URL", "false")
		t.Setenv("TEST_DATABASE_REQUIRED", "true")

		_, _, err := resolveDatabaseURL()
		if err == nil || !strings.Contains(err.Error(), "TEST_DATABASE_REQUIRED") {
			t.Fatalf("resolveDatabaseURL() error = %v, want required-database error", err)
		}
	})

	for _, variable := range []string{"TEST_DATABASE_REQUIRED", "TEST_DATABASE_ALLOW_DATABASE_URL"} {
		t.Run("invalid "+variable+" fails closed", func(t *testing.T) {
			t.Setenv("TEST_DATABASE_URL", "")
			t.Setenv(variable, "sometimes")

			_, _, err := resolveDatabaseURL()
			if err == nil || !strings.Contains(err.Error(), variable) {
				t.Fatalf("resolveDatabaseURL() error = %v, want %s parse error", err, variable)
			}
		})
	}
}

func TestNewSchemaName(t *testing.T) {
	seen := make(map[string]struct{})
	pattern := regexp.MustCompile(`^test_[0-9]+_[0-9a-f]{24}$`)
	for range 100 {
		name, err := newSchemaName()
		if err != nil {
			t.Fatalf("newSchemaName() error = %v", err)
		}
		if !pattern.MatchString(name) {
			t.Fatalf("newSchemaName() = %q, want safe collision-resistant identifier", name)
		}
		if _, duplicate := seen[name]; duplicate {
			t.Fatalf("newSchemaName() returned duplicate %q", name)
		}
		seen[name] = struct{}{}
	}
}

func TestTestPoolConfigUsesDeliberateLimits(t *testing.T) {
	cfg, err := testPoolConfig("postgres://user@localhost/craftsky_test", scopedPoolMax)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxConns != scopedPoolMax || cfg.MinConns != 0 {
		t.Fatalf("pool limits = min %d max %d, want min 0 max %d", cfg.MinConns, cfg.MaxConns, scopedPoolMax)
	}
}

func TestMigrationHelpers(t *testing.T) {
	t.Run("read exact migration", func(t *testing.T) {
		path, err := MigrationPath("000001_bluesky_posts_sample.up.sql")
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(path) != "000001_bluesky_posts_sample.up.sql" {
			t.Fatalf("MigrationPath() = %q", path)
		}
		contents, err := ReadMigration("000001_bluesky_posts_sample.up.sql")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(contents), "CREATE TABLE bluesky_posts_sample") {
			t.Fatal("ReadMigration() did not return exact migration contents")
		}
	})

	t.Run("reject traversal and malformed names", func(t *testing.T) {
		for _, name := range []string{"../go.mod", "000001.sql", "000001_name.txt", "000001_Name.up.sql"} {
			if _, err := MigrationPath(name); err == nil {
				t.Errorf("MigrationPath(%q) error = nil", name)
			}
		}
	})

	t.Run("ordered complete up list", func(t *testing.T) {
		names, err := UpMigrationNames()
		if err != nil {
			t.Fatal(err)
		}
		if len(names) < 2 {
			t.Fatalf("UpMigrationNames() returned %d names", len(names))
		}
		for i, name := range names {
			if !strings.HasSuffix(name, ".up.sql") {
				t.Errorf("name %q is not an up migration", name)
			}
			if i > 0 && names[i-1] >= name {
				t.Fatalf("migration names not strictly ordered: %q then %q", names[i-1], name)
			}
		}
	})
}

func TestWithMigratedSchema(t *testing.T) {
	pool := WithMigratedSchema(t)
	var exists bool
	if err := pool.QueryRow(t.Context(), `SELECT to_regclass('craftsky_posts') IS NOT NULL`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("fully migrated schema does not contain craftsky_posts")
	}
}
