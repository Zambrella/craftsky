package db_test

import (
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

var migrationFilenamePattern = regexp.MustCompile(
	`^([0-9]{6})_(.+)\.(up|down)\.sql$`,
)

func TestMigrationVersionsAreUniqueAndPaired(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob("../../migrations/*.sql")
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no migration files found")
	}

	type migrationPair struct {
		name       string
		directions map[string]string
	}
	versions := make(map[string]migrationPair)
	for _, path := range paths {
		filename := filepath.Base(path)
		match := migrationFilenamePattern.FindStringSubmatch(filename)
		if match == nil {
			t.Errorf("migration filename %q does not follow the versioned convention", filename)
			continue
		}
		version, name, direction := match[1], match[2], match[3]
		pair, exists := versions[version]
		if !exists {
			pair = migrationPair{
				name:       name,
				directions: make(map[string]string, 2),
			}
		}
		if pair.name != name {
			t.Errorf(
				"migration version %s is shared by %q and %q",
				version,
				pair.name,
				name,
			)
		}
		if previous, duplicate := pair.directions[direction]; duplicate {
			t.Errorf(
				"migration version %s has duplicate %s files: %q and %q",
				version,
				direction,
				previous,
				filename,
			)
		}
		pair.directions[direction] = filename
		versions[version] = pair
	}

	sortedVersions := make([]string, 0, len(versions))
	for version := range versions {
		sortedVersions = append(sortedVersions, version)
	}
	sort.Strings(sortedVersions)
	for _, version := range sortedVersions {
		pair := versions[version]
		for _, direction := range []string{"up", "down"} {
			if _, exists := pair.directions[direction]; !exists {
				t.Errorf(
					"migration version %s (%s) has no %s file",
					version,
					pair.name,
					direction,
				)
			}
		}
	}
}
