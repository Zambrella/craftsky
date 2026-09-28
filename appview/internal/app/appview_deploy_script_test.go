package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppViewDeployScriptDeploysExactTaggedCommit(t *testing.T) {
	repo, _, releaseScript := newReleaseTestRepo(t)
	notes := writeNotes(t, "- Ready for production.\n")
	runRelease(t, repo, releaseScript, nil, "create", "appview", "--version", "1.0.4", "--notes", notes)
	runRelease(t, repo, releaseScript, strings.NewReader("y\n"), "push", "appview", "prod-v1.0.4")
	commit := runGit(t, repo, "rev-parse", "HEAD")
	deployScript := repositoryScript(t, "appview-deploy")
	binDir := fakeRenderCurl(t, "live", commit)

	output, err := runDeploy(repo, deployScript, binDir, strings.NewReader("y\n"), commit)
	if err != nil {
		t.Fatalf("deploy: %v\n%s", err, output)
	}
	for _, expected := range []string{
		"AppView deployment succeeded.",
		"Tag: prod-v1.0.4",
		"Commit: " + commit,
		"Render deploy: dep-test",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("deploy output = %q, want %q", output, expected)
		}
	}
}

func TestAppViewDeployScriptRejectsRenderCommitMismatch(t *testing.T) {
	repo, _, releaseScript := newReleaseTestRepo(t)
	notes := writeNotes(t, "- Ready for production.\n")
	runRelease(t, repo, releaseScript, nil, "create", "appview", "--version", "1.0.4", "--notes", notes)
	runRelease(t, repo, releaseScript, strings.NewReader("y\n"), "push", "appview", "prod-v1.0.4")
	commit := runGit(t, repo, "rev-parse", "HEAD")
	deployScript := repositoryScript(t, "appview-deploy")
	binDir := fakeRenderCurl(t, "live", strings.Repeat("a", 40))

	output, err := runDeploy(repo, deployScript, binDir, strings.NewReader("y\n"), commit)
	if err == nil || !strings.Contains(output, "resolved to commit") {
		t.Fatalf("deploy error = %v, output = %q", err, output)
	}
}

func TestAppViewDeployScriptRejectsFailedDeploy(t *testing.T) {
	repo, _, releaseScript := newReleaseTestRepo(t)
	notes := writeNotes(t, "- Ready for production.\n")
	runRelease(t, repo, releaseScript, nil, "create", "appview", "--version", "1.0.4", "--notes", notes)
	runRelease(t, repo, releaseScript, strings.NewReader("y\n"), "push", "appview", "prod-v1.0.4")
	commit := runGit(t, repo, "rev-parse", "HEAD")
	deployScript := repositoryScript(t, "appview-deploy")
	binDir := fakeRenderCurl(t, "build_failed", commit)

	output, err := runDeploy(repo, deployScript, binDir, strings.NewReader("y\n"), commit)
	if err == nil || !strings.Contains(output, "ended with status build_failed") {
		t.Fatalf("deploy error = %v, output = %q", err, output)
	}
}

func TestAppViewDeployScriptRejectsUnhealthyAppView(t *testing.T) {
	repo, _, releaseScript := newReleaseTestRepo(t)
	notes := writeNotes(t, "- Ready for production.\n")
	runRelease(t, repo, releaseScript, nil, "create", "appview", "--version", "1.0.4", "--notes", notes)
	runRelease(t, repo, releaseScript, strings.NewReader("y\n"), "push", "appview", "prod-v1.0.4")
	commit := runGit(t, repo, "rev-parse", "HEAD")
	deployScript := repositoryScript(t, "appview-deploy")
	binDir := fakeRenderCurl(t, "live", commit)
	if err := os.WriteFile(filepath.Join(binDir, "health"), []byte(`{"db":"error","tap":{"connected":false,"last_event_at":""}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	output, err := runDeploy(repo, deployScript, binDir, strings.NewReader("y\n"), commit)
	if err == nil || !strings.Contains(output, "did not reach full production health") {
		t.Fatalf("deploy error = %v, output = %q", err, output)
	}
}

func repositoryScript(t *testing.T, name string) string {
	t.Helper()
	repositoryRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	path, err := filepath.Abs(filepath.Join(repositoryRoot, "scripts", name))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func fakeRenderCurl(t *testing.T, status, deployedCommit string) string {
	t.Helper()
	binDir := t.TempDir()
	path := filepath.Join(binDir, "curl")
	script := `#!/bin/sh
method=GET
data=
auth=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --request) shift; method=$1 ;;
    --data) shift; data=$1 ;;
    --header)
      shift
      [ "$1" != "Authorization: Bearer test-token" ] || auth=true
      ;;
    *) url=$1 ;;
  esac
  shift
done
case "$url" in
  */services/*/deploys)
    [ "$method" = POST ] || { printf 'deploy request was not POST\n' >&2; exit 1; }
    [ "$auth" = true ] || { printf 'deploy request omitted authorization\n' >&2; exit 1; }
    printf '%s' "$data" | jq --exit-status --arg commit "$EXPECTED_COMMIT" '.commitId == $commit and .clearCache == "do_not_clear"' >/dev/null || {
      printf 'deploy request body did not select expected commit\n' >&2
      exit 1
    }
    printf '%s\n' '{"id":"dep-test"}'
    ;;
  */services/*/deploys/dep-test) printf '{"status":"%s","commit":{"id":"%s"}}\n' "$FAKE_RENDER_STATUS" "$FAKE_RENDER_COMMIT" ;;
  */health) printf '%s\n' '{"status":"ok"}' ;;
  */healthz) printf '%s\n' "$FAKE_RENDER_HEALTH" ;;
  *) printf 'unexpected URL: %s\n' "$url" >&2; exit 1 ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "status"), []byte(status+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "commit"), []byte(deployedCommit+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "health"), []byte(`{"db":"ok","tap":{"connected":true,"last_event_at":""}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return binDir
}

func runDeploy(repo, script, binDir string, stdin *strings.Reader, expectedCommit string) (string, error) {
	status, _ := os.ReadFile(filepath.Join(binDir, "status"))
	commit, _ := os.ReadFile(filepath.Join(binDir, "commit"))
	health, _ := os.ReadFile(filepath.Join(binDir, "health"))
	command := exec.Command("/bin/bash", script, "prod-v1.0.4")
	command.Dir = repo
	command.Stdin = stdin
	command.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"RENDER_API_KEY=test-token",
		"RENDER_SERVICE_ID=srv-test",
		"RENDER_API_BASE=https://render.invalid/v1",
		"APPVIEW_PUBLIC_ORIGIN=https://appview.invalid",
		"RENDER_DEPLOY_POLL_ATTEMPTS=1",
		"APPVIEW_HEALTH_ATTEMPTS=1",
		"RENDER_DEPLOY_POLL_SLEEP_SECONDS=0",
		"APPVIEW_HEALTH_SLEEP_SECONDS=0",
		"FAKE_RENDER_STATUS="+strings.TrimSpace(string(status)),
		"FAKE_RENDER_COMMIT="+strings.TrimSpace(string(commit)),
		"FAKE_RENDER_HEALTH="+strings.TrimSpace(string(health)),
		"EXPECTED_COMMIT="+expectedCommit,
	)
	output, err := command.CombinedOutput()
	return string(output), err
}
