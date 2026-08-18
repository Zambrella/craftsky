package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
)

// migrationsDir is the directory used by every subcommand in this family.
// Relative to the CLI's working directory.
const migrationsDir = "migrations"

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply, roll back, or inspect database migrations",
}

// migrateCfg loads only Config (no DB pool). Migrate subcommands use
// golang-migrate's own postgres driver rather than our pgxpool, so we
// don't want loadDeps's side effect of opening a second connection.
// This also means `cli migrate status --env dev` against an empty
// migrations/ directory is validated before configuration is loaded so a
// missing or unreadable operational bundle fails closed.
func migrateCfg() (string, error) {
	env, err := parseEnvFlag()
	if err != nil {
		return "", err
	}
	cfg, err := loadCfgLight(env)
	if err != nil {
		return "", err
	}
	return cfg.DatabaseURL, nil
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all unapplied migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireMigrationsDir(migrationsDir); err != nil {
			return err
		}
		dbURL, err := migrateCfg()
		if err != nil {
			return err
		}
		return runMigrateUp(dbURL, migrationsDir)
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down [N]",
	Short: "Roll back N migrations (default 1)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		n := 1
		if len(args) == 1 {
			v, err := strconv.Atoi(args[0])
			if err != nil || v <= 0 {
				return fmt.Errorf("n must be a positive integer, got %q", args[0])
			}
			n = v
		}
		if err := requireMigrationsDir(migrationsDir); err != nil {
			return err
		}
		dbURL, err := migrateCfg()
		if err != nil {
			return err
		}
		return runMigrateDown(dbURL, migrationsDir, n)
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print current migration version and dirty flag",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireMigrationsDir(migrationsDir); err != nil {
			return err
		}
		dbURL, err := migrateCfg()
		if err != nil {
			return err
		}
		out, err := runMigrateStatus(dbURL, migrationsDir)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

var migrateRedoCmd = &cobra.Command{
	Use:   "redo",
	Short: "Roll back one migration and re-apply it",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireMigrationsDir(migrationsDir); err != nil {
			return err
		}
		dbURL, err := migrateCfg()
		if err != nil {
			return err
		}
		return runMigrateRedo(dbURL, migrationsDir)
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd, migrateDownCmd, migrateStatusCmd, migrateRedoCmd)
	rootCmd.AddCommand(migrateCmd)
}

// inspectMigrationsDir reports whether dir contains no migration SQL files.
// Filesystem errors remain distinguishable from a genuinely empty directory.
func inspectMigrationsDir(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, fmt.Errorf("read migration directory %q: %w", dir, err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			return false, nil
		}
	}
	return true, nil
}

func requireMigrationsDir(dir string) error {
	empty, err := inspectMigrationsDir(dir)
	if err != nil {
		return err
	}
	if empty {
		return fmt.Errorf("migration directory %q is empty", dir)
	}
	return nil
}

// fileSourceURL produces the file:// URL golang-migrate expects. It
// resolves the dir to an absolute path so the URL is well-formed
// regardless of cwd quirks.
func fileSourceURL(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	u := &url.URL{Scheme: "file", Path: abs}
	return u.String(), nil
}

// newMigrate is the shared construction step. Callers pass control.
func newMigrate(databaseURL, dir string) (*migrate.Migrate, error) {
	src, err := fileSourceURL(dir)
	if err != nil {
		return nil, fmt.Errorf("build source url: %w", err)
	}
	m, err := migrate.New(src, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("migrate.New: %w", err)
	}
	return m, nil
}

func runMigrateUp(databaseURL, dir string) error {
	if err := requireMigrationsDir(dir); err != nil {
		return err
	}
	m, err := newMigrate(databaseURL, dir)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func runMigrateDown(databaseURL, dir string, n int) error {
	if err := requireMigrationsDir(dir); err != nil {
		return err
	}
	m, err := newMigrate(databaseURL, dir)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Steps(-n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func runMigrateStatus(databaseURL, dir string) (string, error) {
	if err := requireMigrationsDir(dir); err != nil {
		return "", err
	}
	m, err := newMigrate(databaseURL, dir)
	if err != nil {
		return "", err
	}
	defer m.Close()
	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return "no migrations applied", nil
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("version=%d dirty=%v", v, dirty), nil
}

func runMigrateRedo(databaseURL, dir string) error {
	if err := requireMigrationsDir(dir); err != nil {
		return err
	}
	m, err := newMigrate(databaseURL, dir)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Steps(-1); err != nil {
		return err
	}
	if err := m.Steps(1); err != nil {
		return err
	}
	return nil
}
