import hashlib
from html.parser import HTMLParser
import json
import shutil
import tempfile
import unittest
from pathlib import Path
from urllib.parse import unquote, urlsplit

from cloudflare_pages_build import PUBLIC_FILES, build_site


class CloudflarePagesBuildTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.web = Path(self.temp.name)
        for filename in PUBLIC_FILES:
            (self.web / filename).write_text(f"public {filename}\n")
        (self.web / "assets").mkdir()
        (self.web / "assets" / "logo.svg").write_text("<svg/>")

    def test_build_excludes_development_files_and_removes_stale_output(self):
        for name in ("package.json", "deploy.json", "policy-publication-manifest.json", ".env", "README.md"):
            (self.web / name).write_text("private canary")
        (self.web / "assets" / ".secret.png").write_text("private canary")
        (self.web / "assets" / "notes.txt").write_text("private canary")
        (self.web / "emails").mkdir()
        (self.web / "emails" / "template.html").write_text("private canary")
        (self.web / "dist").mkdir()
        (self.web / "dist" / "stale.html").write_text("old")
        output = build_site(self.web, "abc123", True)
        actual = {path.relative_to(output).as_posix() for path in output.rglob("*") if path.is_file()}
        self.assertEqual(actual, {*PUBLIC_FILES, "assets/logo.svg", "site-release.json"})
        manifest = json.loads((output / "site-release.json").read_text())
        self.assertEqual(manifest["commit"], "abc123")
        self.assertTrue(manifest["dirty"])
        for name, digest in manifest["files"].items():
            self.assertEqual(digest, hashlib.sha256((output / name).read_bytes()).hexdigest())

    # REG-004 / FR-001, FR-012 / AC-001, AC-015.
    def test_status_is_excluded_across_website_build_and_rollback(self):
        (self.web / 'app.json').write_text('status document canary')
        (self.web / 'service-status').mkdir()
        (self.web / 'service-status' / 'app.json').write_text('status document canary')
        for commit in ['website-v2', 'website-v1']:
            output = build_site(self.web, commit, False)
            self.assertFalse((output / 'app.json').exists())
            self.assertFalse((output / 'service-status').exists())
            self.assertEqual((self.web / 'app.json').read_text(), 'status document canary')

    def test_missing_required_page_preserves_previous_build(self):
        output = build_site(self.web, "abc123", False)
        previous = (output / "site-release.json").read_bytes()
        (self.web / "privacy.html").unlink()
        with self.assertRaisesRegex(ValueError, "missing public file"):
            build_site(self.web, "next", False)
        self.assertEqual((output / "site-release.json").read_bytes(), previous)

    def test_symlink_cannot_publish_a_file_outside_site(self):
        secret = self.web / "secret"
        secret.write_text("private canary")
        (self.web / "assets" / "leak.png").symlink_to(secret)
        with self.assertRaisesRegex(ValueError, "symlink"):
            build_site(self.web, "abc123", False)

    def test_policy_drafts_are_copied_without_changing_approval(self):
        (self.web / "privacy.html").write_text("Draft; not effective")
        output = build_site(self.web, "abc123", False)
        self.assertEqual((output / "privacy.html").read_text(), "Draft; not effective")


class WebsiteLinks(HTMLParser):
    def __init__(self):
        super().__init__()
        self.targets = []

    def handle_starttag(self, tag, attrs):
        if tag not in {"a", "link", "script", "img"}:
            return
        for name, value in attrs:
            if name in {"href", "src"} and value:
                self.targets.append(value)


class RepositoryWebsiteSmokeTests(unittest.TestCase):
    def test_built_pages_link_to_published_local_files(self):
        source = Path(__file__).resolve().parents[1] / "web"
        with tempfile.TemporaryDirectory() as temporary:
            web = Path(temporary) / "web"
            web.mkdir()
            for name in PUBLIC_FILES:
                shutil.copy2(source / name, web / name)
            shutil.copytree(source / "assets", web / "assets")
            output = build_site(web, "fixture", False)
            for page in output.glob("*.html"):
                links = WebsiteLinks()
                links.feed(page.read_text())
                for target in links.targets:
                    url = urlsplit(target)
                    if url.scheme or url.netloc or not url.path:
                        continue
                    path = unquote(url.path).lstrip("/")
                    file = output / path if url.path.startswith("/") else page.parent / path
                    candidates = [file / "index.html", file, file.with_suffix(".html")]
                    with self.subTest(page=page.name, target=target):
                        self.assertTrue(
                            any(candidate.is_file() for candidate in candidates),
                            f"{page.name} links to unpublished file: {target}",
                        )


if __name__ == "__main__":
    unittest.main()
