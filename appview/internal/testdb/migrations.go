package testdb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

var migrationNamePattern = regexp.MustCompile(`^[0-9]{6}_[a-z0-9_]+\.(up|down)\.sql$`)

// MigrationExecutor is implemented by pgx pools, connections, and
// transactions, allowing migration tests to apply exact SQL at the boundary
// they need to exercise.
type MigrationExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// MigrationPath returns the repository migration path for an exact migration
// filename. Names with path separators or non-migration suffixes are rejected.
func MigrationPath(name string) (string, error) {
	if filepath.Base(name) != name || !migrationNamePattern.MatchString(name) {
		return "", fmt.Errorf("invalid migration filename %q", name)
	}
	path := filepath.Join(migrationsDirectory(), name)
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("locate migration %q: %w", name, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("migration %q is not a regular file", name)
	}
	return path, nil
}

// ReadMigration reads one exact production migration without transformation.
func ReadMigration(name string) ([]byte, error) {
	path, err := MigrationPath(name)
	if err != nil {
		return nil, err
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read migration %q: %w", name, err)
	}
	return contents, nil
}

// UpMigrationNames returns every production up migration in version order.
func UpMigrationNames() ([]string, error) {
	entries, err := os.ReadDir(migrationsDirectory())
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	names := make([]string, 0, len(entries)/2)
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		if !migrationNamePattern.MatchString(entry.Name()) {
			return nil, fmt.Errorf("invalid up migration filename %q", entry.Name())
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("no up migrations found in %s", migrationsDirectory())
	}
	for i := 1; i < len(names); i++ {
		if names[i-1][:6] == names[i][:6] {
			return nil, fmt.Errorf("duplicate migration version %s: %q and %q", names[i][:6], names[i-1], names[i])
		}
	}
	return names, nil
}

// ApplyMigrations executes the named production migration files exactly as
// stored. The caller controls ordering so historical pre-state tests can apply
// a deliberate subset.
func ApplyMigrations(ctx context.Context, executor MigrationExecutor, names ...string) error {
	for _, name := range names {
		contents, err := ReadMigration(name)
		if err != nil {
			return err
		}
		if _, err := executor.Exec(ctx, string(contents)); err != nil {
			return fmt.Errorf("apply migration %q: %w", name, err)
		}
	}
	return nil
}

// ApplyAllUpMigrations applies every exact production up migration in order.
func ApplyAllUpMigrations(ctx context.Context, executor MigrationExecutor) error {
	names, err := UpMigrationNames()
	if err != nil {
		return err
	}
	return ApplyMigrations(ctx, executor, names...)
}
