package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseScriptCreatesAppViewRelease(t *testing.T) {
	repo, remote, script := newReleaseTestRepo(t)
	writeTestFile(t, repo, "appview/feature.txt", "first generated notes fixture\n")
	runGit(t, repo, "add", "appview/feature.txt")
	runGit(t, repo, "commit", "-m", "feat: add generated release notes")
	writeTestFile(t, repo, "app/unrelated.txt", "app-only change\n")
	runGit(t, repo, "add", "app/unrelated.txt")
	runGit(t, repo, "commit", "-m", "feat: app-only change")
	writeTestFile(t, repo, "appview/feature.txt", "second generated notes fixture\n")
	runGit(t, repo, "add", "appview/feature.txt")
	runGit(t, repo, "commit", "-m", "fix: preserve generated note order")
	runGit(t, repo, "push", "origin", "main")

	output := runRelease(t, repo, script, nil, "create", "appview", "--version", "1.0.4")
	if !strings.Contains(output, "Created local release prod-v1.0.4") {
		t.Fatalf("create output = %q", output)
	}
	if got := strings.TrimSpace(mustReadFile(t, filepath.Join(repo, "appview", "VERSION"))); got != "1.0.4" {
		t.Fatalf("VERSION = %q", got)
	}
	changelog := mustReadFile(t, filepath.Join(repo, "appview", "CHANGELOG.md"))
	wantChangelog := "# AppView Changelog\n\n## 1.0.4 - 2026-09-17\n\n- feat: add generated release notes\n- fix: preserve generated note order\n\nRelease history before local release automation is represented by Git tags.\n"
	if changelog != wantChangelog {
		t.Fatalf("changelog = %q, want %q", changelog, wantChangelog)
	}
	if got := runGit(t, repo, "log", "-1", "--format=%s"); got != "chore(release): AppView 1.0.4" {
		t.Fatalf("commit subject = %q", got)
	}
	if got := runGit(t, repo, "cat-file", "-t", "refs/tags/prod-v1.0.4"); got != "tag" {
		t.Fatalf("tag type = %q", got)
	}
	if got := runGit(t, repo, "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD"); got != "appview/CHANGELOG.md\nappview/VERSION" {
		t.Fatalf("release files = %q", got)
	}
	if gitRefExists(remote, "refs/tags/prod-v1.0.4") {
		t.Fatal("release tag was unexpectedly pushed")
	}
}

func TestReleaseScriptCreatesAppReleaseWithHigherBuild(t *testing.T) {
	repo, _, script := newReleaseTestRepo(t)
	notes := writeNotes(t, "- Prepared the mobile stores.\n- Preserved curated wording.\n")

	runRelease(t, repo, script, nil, "create", "app", "--version", "1.1.0+2", "--notes", notes)
	pubspec := mustReadFile(t, filepath.Join(repo, "app", "pubspec.yaml"))
	if !strings.Contains(pubspec, "version: 1.1.0+2") {
		t.Fatalf("pubspec = %q", pubspec)
	}
	if got := runGit(t, repo, "tag", "--points-at", "HEAD"); got != "app-v1.1.0+2" {
		t.Fatalf("tag = %q", got)
	}
	if got := runGit(t, repo, "cat-file", "-t", "refs/tags/app-v1.1.0+2"); got != "tag" {
		t.Fatalf("tag type = %q", got)
	}
	if got := runGit(t, repo, "log", "-1", "--format=%s"); got != "chore(release): app 1.1.0+2" {
		t.Fatalf("commit subject = %q", got)
	}
	wantChangelog := "# App Changelog\n\n## 1.1.0+2 - 2026-09-17\n\n- Prepared the mobile stores.\n- Preserved curated wording.\n\nRelease history before local release automation is represented by Git tags.\n"
	if got := mustReadFile(t, filepath.Join(repo, "app", "CHANGELOG.md")); got != wantChangelog {
		t.Fatalf("changelog = %q, want %q", got, wantChangelog)
	}
	if got := runGit(t, repo, "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD"); got != "app/CHANGELOG.md\napp/pubspec.yaml" {
		t.Fatalf("release files = %q", got)
	}
}

func TestReleaseScriptRejectsInvalidCreateWithoutMutation(t *testing.T) {
	for _, test := range []struct {
		name    string
		stream  string
		version string
		notes   string
		want    string
	}{
		{name: "non-increasing appview", stream: "appview", version: "1.0.2", notes: "valid", want: "must be greater than 1.0.3"},
		{name: "lower app build", stream: "app", version: "1.1.0+1", notes: "valid", want: "build number must be greater"},
		{name: "unchanged app marketing version", stream: "app", version: "1.0.0+2", notes: "valid", want: "build number must be greater"},
		{name: "malformed appview", stream: "appview", version: "1.0", notes: "valid", want: "invalid AppView version"},
		{name: "malformed app", stream: "app", version: "1.0.1", notes: "valid", want: "invalid app version"},
		{name: "empty notes", stream: "appview", version: "1.0.4", notes: "", want: "nonempty readable file"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, _, script := newReleaseTestRepo(t)
			notes := writeNotes(t, test.notes)
			before := runGit(t, repo, "rev-parse", "HEAD")
			output, err := runReleaseError(repo, script, nil, "create", test.stream, "--version", test.version, "--notes", notes)
			if err == nil || !strings.Contains(output, test.want) {
				t.Fatalf("create error = %v, output = %q, want %q", err, output, test.want)
			}
			if got := runGit(t, repo, "rev-parse", "HEAD"); got != before {
				t.Fatalf("HEAD changed from %s to %s", before, got)
			}
			if got := runGit(t, repo, "status", "--porcelain"); got != "" {
				t.Fatalf("working tree changed: %q", got)
			}
		})
	}

	t.Run("no relevant generated notes", func(t *testing.T) {
		repo, _, script := newReleaseTestRepo(t)
		before := runGit(t, repo, "rev-parse", "HEAD")
		output, err := runReleaseError(repo, script, nil, "create", "appview", "--version", "1.0.4")
		if err == nil || !strings.Contains(output, "no relevant commits since prod-v1.0.3") {
			t.Fatalf("create error = %v, output = %q", err, output)
		}
		if got := runGit(t, repo, "rev-parse", "HEAD"); got != before {
			t.Fatalf("HEAD changed from %s to %s", before, got)
		}
		if got := runGit(t, repo, "status", "--porcelain"); got != "" {
			t.Fatalf("working tree changed: %q", got)
		}
	})

	t.Run("missing manual notes", func(t *testing.T) {
		repo, _, script := newReleaseTestRepo(t)
		before := runGit(t, repo, "rev-parse", "HEAD")
		output, err := runReleaseError(
			repo,
			script,
			nil,
			"create",
			"appview",
			"--version",
			"1.0.4",
			"--notes",
			filepath.Join(t.TempDir(), "missing.md"),
		)
		if err == nil || !strings.Contains(output, "nonempty readable file") {
			t.Fatalf("create error = %v, output = %q", err, output)
		}
		if got := runGit(t, repo, "rev-parse", "HEAD"); got != before {
			t.Fatalf("HEAD changed from %s to %s", before, got)
		}
		if got := runGit(t, repo, "status", "--porcelain"); got != "" {
			t.Fatalf("working tree changed: %q", got)
		}
	})
}

func TestReleaseScriptRejectsCreatePreflightFailuresWithoutMutation(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, string, string)
		want  string
	}{
		{name: "dirty tree", setup: func(t *testing.T, repo, _ string) { writeTestFile(t, repo, "dirty.txt", "dirty\n") }, want: "working tree must be clean"},
		{name: "wrong branch", setup: func(t *testing.T, repo, _ string) { runGit(t, repo, "switch", "-c", "release-test") }, want: "release commands require main"},
		{name: "stale main", setup: func(t *testing.T, repo, _ string) { runGit(t, repo, "commit", "--allow-empty", "-m", "local advance") }, want: "not the current origin/main"},
		{name: "local tag exists", setup: func(t *testing.T, repo, _ string) { runGit(t, repo, "tag", "prod-v1.0.4") }, want: "local tag already exists"},
		{name: "remote tag exists", setup: func(t *testing.T, repo, _ string) {
			runGit(t, repo, "tag", "prod-v1.0.4")
			runGit(t, repo, "push", "origin", "prod-v1.0.4")
			runGit(t, repo, "tag", "-d", "prod-v1.0.4")
		}, want: "local tag already exists"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, _, script := newReleaseTestRepo(t)
			test.setup(t, repo, script)
			before := runGit(t, repo, "rev-parse", "HEAD")
			notes := writeNotes(t, "- Valid notes.\n")
			output, err := runReleaseError(repo, script, nil, "create", "appview", "--version", "1.0.4", "--notes", notes)
			if err == nil || !strings.Contains(output, test.want) {
				t.Fatalf("create error = %v, output = %q, want %q", err, output, test.want)
			}
			if got := runGit(t, repo, "rev-parse", "HEAD"); got != before {
				t.Fatalf("HEAD changed from %s to %s", before, got)
			}
		})
	}
}

func TestReleaseScriptPushesCommitAndTagAtomically(t *testing.T) {
	repo, remote, script := newReleaseTestRepo(t)
	notes := writeNotes(t, "- Ready for production.\n")
	runRelease(t, repo, script, nil, "create", "appview", "--version", "1.0.4", "--notes", notes)
	releaseCommit := runGit(t, repo, "rev-parse", "HEAD")

	output := runRelease(t, repo, script, strings.NewReader("y\n"), "push", "appview", "prod-v1.0.4")
	if !strings.Contains(output, "-> main") {
		t.Fatalf("push output = %q", output)
	}
	if got := runGit(t, "", "--git-dir", remote, "rev-parse", "refs/heads/main"); got != releaseCommit {
		t.Fatalf("remote main = %q, want %q", got, releaseCommit)
	}
	if got := runGit(t, "", "--git-dir", remote, "rev-parse", "refs/tags/prod-v1.0.4^{}"); got != releaseCommit {
		t.Fatalf("remote tag = %q, want %q", got, releaseCommit)
	}
}

func TestReleaseScriptRejectsPushCommitWithExtraFiles(t *testing.T) {
	repo, remote, script := newReleaseTestRepo(t)
	notes := writeNotes(t, "- Ready for production.\n")
	runRelease(t, repo, script, nil, "create", "appview", "--version", "1.0.4", "--notes", notes)
	writeTestFile(t, repo, "unrelated.txt", "must not ship in a release commit\n")
	runGit(t, repo, "add", "unrelated.txt")
	runGit(t, repo, "commit", "--amend", "--no-edit")
	runGit(t, repo, "tag", "-d", "prod-v1.0.4")
	runGit(t, repo, "tag", "-a", "prod-v1.0.4", "-m", "AppView 1.0.4")

	output, err := runReleaseError(repo, script, strings.NewReader("y\n"), "push", "appview", "prod-v1.0.4")
	if err == nil || !strings.Contains(output, "release commit must change only") {
		t.Fatalf("push error = %v, output = %q", err, output)
	}
	if gitRefExists(remote, "refs/tags/prod-v1.0.4") {
		t.Fatal("release tag was unexpectedly pushed")
	}
}

func TestReleaseScriptRejectsHeadChangeWhileAwaitingPushConfirmation(t *testing.T) {
	repo, remote, script := newReleaseTestRepo(t)
	notes := writeNotes(t, "- Ready for production.\n")
	runRelease(t, repo, script, nil, "create", "appview", "--version", "1.0.4", "--notes", notes)
	binDir := t.TempDir()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "fetch-count")
	wrapper := filepath.Join(binDir, "git")
	wrapperScript := `#!/bin/bash
set -euo pipefail
if [[ "${1:-}" == fetch && "${2:-}" == origin && "${3:-}" == +refs/heads/main:refs/remotes/origin/main ]]; then
  count=0
  [[ ! -f "$FETCH_STATE" ]] || count=$(cat "$FETCH_STATE")
  count=$((count + 1))
  printf '%s\n' "$count" >"$FETCH_STATE"
  if [[ "$count" == 2 ]]; then
    "$REAL_GIT" commit --allow-empty -m "concurrent local commit" >/dev/null
  fi
fi
exec "$REAL_GIT" "$@"
`
	if err := os.WriteFile(wrapper, []byte(wrapperScript), 0o755); err != nil {
		t.Fatal(err)
	}
	env := []string{
		"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"REAL_GIT=" + realGit,
		"FETCH_STATE=" + state,
	}
	output, err := runReleaseErrorWithEnv(
		repo,
		script,
		strings.NewReader("y\n"),
		[]string{"push", "appview", "prod-v1.0.4"},
		env,
	)
	if err == nil || !strings.Contains(output, "HEAD changed while awaiting confirmation") {
		t.Fatalf("push error = %v, output = %q", err, output)
	}
	if gitRefExists(remote, "refs/tags/prod-v1.0.4") {
		t.Fatal("release tag was unexpectedly pushed")
	}
}

func TestReleaseScriptRejectsPushWhenOriginMainAdvanced(t *testing.T) {
	repo, remote, script := newReleaseTestRepo(t)
	notes := writeNotes(t, "- Ready for production.\n")
	runRelease(t, repo, script, nil, "create", "appview", "--version", "1.0.4", "--notes", notes)

	other := t.TempDir()
	runGit(t, "", "clone", remote, other)
	runGit(t, other, "config", "user.name", "Other Developer")
	runGit(t, other, "config", "user.email", "other@example.invalid")
	writeTestFile(t, other, "other.txt", "advanced\n")
	runGit(t, other, "add", ".")
	runGit(t, other, "commit", "-m", "advance main")

	binDir := t.TempDir()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "fetch-count")
	wrapper := filepath.Join(binDir, "git")
	wrapperScript := `#!/bin/bash
set -euo pipefail
if [[ "${1:-}" == fetch && "${2:-}" == origin && "${3:-}" == +refs/heads/main:refs/remotes/origin/main ]]; then
  count=0
  [[ ! -f "$FETCH_STATE" ]] || count=$(cat "$FETCH_STATE")
  count=$((count + 1))
  printf '%s\n' "$count" >"$FETCH_STATE"
  if [[ "$count" == 2 ]]; then
    "$REAL_GIT" -C "$OTHER_REPO" push origin main >/dev/null
  fi
fi
exec "$REAL_GIT" "$@"
`
	if err := os.WriteFile(wrapper, []byte(wrapperScript), 0o755); err != nil {
		t.Fatal(err)
	}
	env := []string{
		"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"REAL_GIT=" + realGit,
		"FETCH_STATE=" + state,
		"OTHER_REPO=" + other,
	}
	output, err := runReleaseErrorWithEnv(
		repo,
		script,
		strings.NewReader("y\n"),
		[]string{"push", "appview", "prod-v1.0.4"},
		env,
	)
	if err == nil || !strings.Contains(output, "origin/main changed while awaiting confirmation") {
		t.Fatalf("push error = %v, output = %q", err, output)
	}
	if gitRefExists(remote, "refs/tags/prod-v1.0.4") {
		t.Fatal("release tag was unexpectedly pushed")
	}
}

func newReleaseTestRepo(t *testing.T) (string, string, string) {
	t.Helper()
	repositoryRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	script, err := filepath.Abs(filepath.Join(repositoryRoot, "scripts", "release"))
	if err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Release Test")
	runGit(t, repo, "config", "user.email", "release@example.invalid")
	writeTestFile(t, repo, "appview/VERSION", "1.0.3\n")
	writeTestFile(t, repo, "appview/CHANGELOG.md", "# AppView Changelog\n\nRelease history before local release automation is represented by Git tags.\n")
	writeTestFile(t, repo, "app/pubspec.yaml", "name: test\nversion: 1.0.0+1\n")
	writeTestFile(t, repo, "app/CHANGELOG.md", "# App Changelog\n\nRelease history before local release automation is represented by Git tags.\n")
	writeTestFile(t, repo, ".gitignore", "app/config/*.env\napp/android/key.properties\napp/android/app/*.jks\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "baseline")
	runGit(t, repo, "tag", "prod-v1.0.3")
	runGit(t, repo, "tag", "app-v1.0.0+1")
	remote := filepath.Join(t.TempDir(), "origin.git")
	runGit(t, "", "init", "--bare", remote)
	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "push", "--tags", "-u", "origin", "main")
	return repo, remote, script
}

func writeNotes(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func runRelease(t *testing.T, repo, script string, stdin *strings.Reader, args ...string) string {
	t.Helper()
	output, err := runReleaseError(repo, script, stdin, args...)
	if err != nil {
		t.Fatalf("release %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return output
}

func runReleaseError(repo, script string, stdin *strings.Reader, args ...string) (string, error) {
	return runReleaseErrorWithEnv(repo, script, stdin, args, nil)
}

func runReleaseErrorWithEnv(repo, script string, stdin *strings.Reader, args, extraEnv []string) (string, error) {
	command := exec.Command("/bin/bash", append([]string{script}, args...)...)
	command.Dir = repo
	command.Env = append(os.Environ(), append([]string{"RELEASE_DATE=2026-09-17"}, extraEnv...)...)
	if stdin != nil {
		command.Stdin = stdin
	}
	output, err := command.CombinedOutput()
	return string(output), err
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func gitRefExists(gitDir, ref string) bool {
	command := exec.Command("git", "--git-dir", gitDir, "show-ref", "--verify", "--quiet", ref)
	return command.Run() == nil
}

func writeTestFile(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
