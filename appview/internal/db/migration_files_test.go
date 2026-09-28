package db_test

import (
	"regexp"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

var migrationFilenamePattern = regexp.MustCompile(
	`^([0-9]{6})_(.+)\.(up|down)\.sql$`,
)

func TestMigrationVersionsAreUniqueAndPaired(t *testing.T) {
	t.Parallel()

	names, err := testdb.UpMigrationNames()
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	versions := make(map[string]string, len(names))
	for _, filename := range names {
		match := migrationFilenamePattern.FindStringSubmatch(filename)
		if match == nil {
			t.Errorf("migration filename %q does not follow the versioned convention", filename)
			continue
		}
		version, migrationName := match[1], match[2]
		if previous, duplicate := versions[version]; duplicate {
			t.Errorf("migration version %s is shared by %q and %q", version, previous, migrationName)
		}
		versions[version] = migrationName
		downName := version + "_" + migrationName + ".down.sql"
		if _, err := testdb.MigrationPath(downName); err != nil {
			t.Errorf("migration version %s (%s) has no down file: %v", version, migrationName, err)
		}
	}
}
