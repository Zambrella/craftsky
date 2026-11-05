"""Exercise release creation, atomic pushes and upload gates against local Git remotes."""
import importlib.machinery
import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SCRIPTS = Path(__file__).resolve().parent
loader = importlib.machinery.SourceFileLoader("release_deploy", str(SCRIPTS / "web-client-deploy"))
spec = importlib.util.spec_from_loader(loader.name, loader)
deploy = importlib.util.module_from_spec(spec)
loader.exec_module(deploy)


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        root = Path(self.temporary.name)
        self.repo = root / "repo"
        self.repo.mkdir()
        self.run_git("init", "--bare", str(root / "origin.git"))
        self.run_git("init", "-b", "main")
        self.run_git("config", "user.email", "release-test@example.invalid")
        self.run_git("config", "user.name", "Release Test")
        self.run_git("config", "commit.gpgsign", "false")
        self.run_git("config", "tag.gpgsign", "false")
        for name, content in {"web-client/VERSION": "1.11.1\n",
                              "appview/VERSION": "1.2.0\n",
                              "app/pubspec.yaml": "version: 1.11.1+17\n"}.items():
            path = self.repo / name
            path.parent.mkdir(exist_ok=True)
            path.write_text(content)
        (self.repo / "notes.md").write_text("- Reviewed release.\n")
        self.run_git("add", ".")
        self.run_git("commit", "-m", "initial")
        self.run_git("remote", "add", "origin", str(root / "origin.git"))
        self.run_git("push", "origin", "main")

    def run_git(self, *args):
        return subprocess.check_output(["git", *args], cwd=self.repo, text=True, stderr=subprocess.PIPE).strip()

    def release(self, *args, success=True):
        result = subprocess.run(["bash", str(SCRIPTS / "release"), *args], cwd=self.repo,
                                text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        if success:
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        else:
            self.assertNotEqual(result.returncode, 0, result.stdout)
        return result

    def create_web(self):
        self.release("create", "web", "--version", "1.11.2", "--notes", "notes.md")

    def test_web_release_creation_push_and_upload_identity(self):
        self.create_web()
        self.assertEqual(self.run_git("cat-file", "-t", "web-v1.11.2"), "tag")
        self.assertEqual((self.repo / "web-client/VERSION").read_text(), "1.11.2\n")
        self.assertIn("## 1.11.2 - ", (self.repo / "web-client/CHANGELOG.md").read_text())
        with patch.object(deploy, "ROOT", self.repo):
            with self.assertRaisesRegex(ValueError, "pushed unchanged"):
                deploy.release_commit("web-v1.11.2")
            self.release("push", "web", "web-v1.11.2")
            self.assertEqual(deploy.release_commit("web-v1.11.2"), self.run_git("rev-parse", "HEAD"))
            (self.repo / "notes.md").write_text("uncommitted\n")
            with self.assertRaisesRegex(ValueError, "working tree must be clean"):
                deploy.release_commit("web-v1.11.2")

    def test_web_version_must_increase_and_first_release_requires_notes(self):
        result = self.release("create", "web", "--version", "1.11.1", "--notes", "notes.md", success=False)
        self.assertIn("must be greater", result.stderr)
        result = self.release("create", "web", "--version", "1.11.2", success=False)
        self.assertIn("provide --notes", result.stderr)

    def test_later_web_release_generates_notes_and_keeps_history(self):
        self.create_web()
        self.release("push", "web", "web-v1.11.2")
        (self.repo / "web-client/feature.txt").write_text("feature\n")
        self.run_git("add", ".")
        self.run_git("commit", "-m", "fix: web feature")
        self.run_git("push", "origin", "main")
        self.release("create", "web", "--version", "1.11.3")
        changelog = (self.repo / "web-client/CHANGELOG.md").read_text()
        self.assertIn("- fix: web feature", changelog)
        self.assertLess(changelog.index("## 1.11.3"), changelog.index("## 1.11.2"))
        self.release("push", "web", "web-v1.11.3")

    def test_lightweight_and_mismatched_tags_are_rejected(self):
        self.run_git("tag", "web-v1.11.1")
        with patch.object(deploy, "ROOT", self.repo):
            with self.assertRaisesRegex(ValueError, "annotated"):
                deploy.release_commit("web-v1.11.1")
            self.run_git("tag", "-a", "web-v1.11.2", "-m", "wrong version")
            self.run_git("push", "origin", "web-v1.11.2")
            with self.assertRaisesRegex(ValueError, "does not match"):
                deploy.release_commit("web-v1.11.2")

    def test_later_commits_and_non_main_uploads_are_rejected(self):
        self.create_web()
        self.release("push", "web", "web-v1.11.2")
        with patch.object(deploy, "ROOT", self.repo):
            self.run_git("checkout", "-b", "feature")
            with self.assertRaisesRegex(ValueError, "require main"):
                deploy.release_commit("web-v1.11.2")
            self.run_git("checkout", "main")
            self.run_git("commit", "--allow-empty", "-m", "later change")
            with self.assertRaisesRegex(ValueError, "point to HEAD"):
                deploy.release_commit("web-v1.11.2")

    def test_release_push_rejects_additional_files(self):
        self.create_web()
        (self.repo / "notes.md").write_text("extra change\n")
        self.run_git("add", "notes.md")
        self.run_git("commit", "--amend", "--no-edit")
        self.run_git("tag", "-f", "-a", "web-v1.11.2", "-m", "release")
        result = self.release("push", "web", "web-v1.11.2", success=False)
        self.assertIn("must change only", result.stderr)

    def test_existing_release_streams_still_create_and_push(self):
        for stream, version, tag in [("appview", "1.2.1", "prod-v1.2.1"),
                                     ("app", "1.11.2+18", "app-v1.11.2+18")]:
            with self.subTest(stream=stream):
                self.release("create", stream, "--version", version, "--notes", "notes.md")
                self.release("push", stream, tag)


if __name__ == "__main__":
    unittest.main()
