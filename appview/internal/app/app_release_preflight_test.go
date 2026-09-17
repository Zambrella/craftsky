package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppReleasePreflightRunsFlutterChecksForTaggedHead(t *testing.T) {
	repo, _, releaseScript := newReleaseTestRepo(t)
	notes := writeNotes(t, "- Ready for mobile stores.\n")
	runRelease(t, repo, releaseScript, nil, "create", "app", "--version", "1.1.0+2", "--notes", notes)
	preflight := repositoryScript(t, "app-release-preflight")
	binDir, logPath := prepareAppReleaseEnvironment(t, repo, false)

	output, err := runAppReleasePreflight(repo, preflight, binDir, nil)
	if err != nil {
		t.Fatalf("preflight: %v\n%s", err, output)
	}
	if got := mustReadFile(t, logPath); got != "pub get\nanalyze\ntest\n" {
		t.Fatalf("flutter calls = %q", got)
	}
}

func TestAppReleasePreflightRejectsMissingSentryCredentials(t *testing.T) {
	repo, _, releaseScript := newReleaseTestRepo(t)
	notes := writeNotes(t, "- Ready for mobile stores.\n")
	runRelease(t, repo, releaseScript, nil, "create", "app", "--version", "1.1.0+2", "--notes", notes)
	preflight := repositoryScript(t, "app-release-preflight")
	binDir, _ := prepareAppReleaseEnvironment(t, repo, false)

	output, err := runAppReleasePreflight(repo, preflight, binDir, []string{"SENTRY_AUTH_TOKEN="})
	if err == nil || !strings.Contains(string(output), "Set SENTRY_AUTH_TOKEN") {
		t.Fatalf("preflight error = %v, output = %q", err, output)
	}
}

func TestAppReleasePreflightRejectsMissingAndroidSigningMaterial(t *testing.T) {
	repo, _, releaseScript := newReleaseTestRepo(t)
	notes := writeNotes(t, "- Ready for mobile stores.\n")
	runRelease(t, repo, releaseScript, nil, "create", "app", "--version", "1.1.0+2", "--notes", notes)
	preflight := repositoryScript(t, "app-release-preflight")
	binDir, _ := prepareAppReleaseEnvironment(t, repo, false)
	if err := os.Remove(filepath.Join(repo, "app", "android", "app", "release.jks")); err != nil {
		t.Fatal(err)
	}

	output, err := runAppReleasePreflight(repo, preflight, binDir, nil)
	if err == nil || !strings.Contains(string(output), "Android release keystore not found") {
		t.Fatalf("preflight error = %v, output = %q", err, output)
	}
}

func TestAppReleasePreflightPropagatesFlutterFailure(t *testing.T) {
	repo, _, releaseScript := newReleaseTestRepo(t)
	notes := writeNotes(t, "- Ready for mobile stores.\n")
	runRelease(t, repo, releaseScript, nil, "create", "app", "--version", "1.1.0+2", "--notes", notes)
	preflight := repositoryScript(t, "app-release-preflight")
	binDir, _ := prepareAppReleaseEnvironment(t, repo, true)

	output, err := runAppReleasePreflight(repo, preflight, binDir, nil)
	if err == nil || !strings.Contains(string(output), "analyze failed") {
		t.Fatalf("preflight error = %v, output = %q", err, output)
	}
}

func prepareAppReleaseEnvironment(t *testing.T, repo string, failAnalyze bool) (string, string) {
	t.Helper()
	writeTestFile(t, repo, "app/config/production.env", "SENTRY_DSN=https://public@example.invalid/1\nSENTRY_ENVIRONMENT=production\n")
	writeTestFile(t, repo, "app/android/key.properties", "keyAlias=release\nkeyPassword=test\nstoreFile=release.jks\nstorePassword=test\n")
	writeTestFile(t, repo, "app/android/app/release.jks", "test keystore\n")
	binDir := t.TempDir()
	logPath := filepath.Join(binDir, "flutter.log")
	failure := ""
	if failAnalyze {
		failure = "[ \"$1\" != analyze ] || { printf 'analyze failed\\n' >&2; exit 1; }\n"
	}
	fakeFlutter := filepath.Join(binDir, "flutter")
	contents := "#!/bin/sh\nprintf '%s\\n' \"$*\" >>\"$FAKE_FLUTTER_LOG\"\n" + failure
	if err := os.WriteFile(fakeFlutter, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
	return binDir, logPath
}

func runAppReleasePreflight(repo, preflight, binDir string, extraEnv []string) ([]byte, error) {
	command := exec.Command("/bin/bash", preflight, "appbundle", "app/config/production.env")
	command.Dir = repo
	logPath := filepath.Join(binDir, "flutter.log")
	command.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_FLUTTER_LOG="+logPath,
		"SENTRY_AUTH_TOKEN=test-token",
		"SENTRY_ORG=test-org",
		"SENTRY_PROJECT=test-project",
	)
	command.Env = append(command.Env, extraEnv...)
	return command.CombinedOutput()
}
