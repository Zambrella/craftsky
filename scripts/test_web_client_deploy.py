"""Safety boundaries for the preview-only release command."""
import importlib.machinery
import importlib.util
import json
import io
from urllib.error import HTTPError
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import MagicMock, patch

loader = importlib.machinery.SourceFileLoader("web_client_deploy", str(Path(__file__).with_name("web-client-deploy")))
spec = importlib.util.spec_from_loader(loader.name, loader)
deploy = importlib.util.module_from_spec(spec)
loader.exec_module(deploy)


class PreviewTests(unittest.TestCase):
    def test_invalid_or_dirty_release_never_uploads(self):
        for tag, status, message in [("not-a-release", "", "web release tag"),
                                     ("web-v1.2.3", " M file", "working tree must be clean")]:
            with self.subTest(tag=tag), patch.object(deploy, "git", return_value=status), \
                    patch.object(deploy.subprocess, "run") as run, \
                    patch("sys.argv", ["web-client-deploy", "preview", tag]):
                with self.assertRaisesRegex(ValueError, message):
                    deploy.main()
                run.assert_not_called()

    def test_failed_checks_never_upload(self):
        with patch.object(deploy, "release_commit", return_value="a" * 40), \
                patch.object(deploy.subprocess, "run", side_effect=[None, subprocess.CalledProcessError(1, "just")]) as run, \
                patch("sys.argv", ["web-client-deploy", "preview", "web-v1.2.3"]):
            with self.assertRaises(subprocess.CalledProcessError):
                deploy.main()
            self.assertEqual(run.call_count, 2)
            self.assertEqual(run.call_args_list[-1].args[0], ["just", "web-client-check"])

    def test_upload_records_exact_version_and_never_deploys(self):
        with tempfile.TemporaryDirectory() as temporary:
            client = Path(temporary)
            artifact = client / ".svelte-kit/cloudflare"
            artifact.mkdir(parents=True)
            (artifact / "_worker.js").write_text("export default {}")
            maps = client / ".sentry-artifacts"
            maps.mkdir()
            (maps / "release.json").write_text(json.dumps({"release": "craftsky-web@1.2.3+aaaaaaaaaaaa"}))
            commands = []

            def run(command, **kwargs):
                commands.append(command)
                if "upload" in command:
                    Path(kwargs["env"]["WRANGLER_OUTPUT_FILE_PATH"]).write_text(json.dumps({
                        "type": "version-upload", "worker_name": deploy.WORKER,
                        "version_id": "version-123", "preview_url": "https://version.example.workers.dev",
                    }) + "\n")

            with patch.object(deploy, "CLIENT", client), \
                    patch.object(deploy, "release_commit", return_value="a" * 40), \
                    patch.object(deploy.subprocess, "run", side_effect=run), \
                    patch.object(deploy, "verify") as verify, \
                    patch("sys.argv", ["web-client-deploy", "preview", "web-v1.2.3"]):
                self.assertEqual(deploy.main(), 0)
            upload = commands[-1]
            self.assertEqual(upload[2:4], ["versions", "upload"])
            self.assertIn("--preview-alias", upload)
            self.assertEqual(upload[upload.index("--tag") + 1], "web-v1.2.3")
            self.assertNotIn("deploy", upload)
            receipt = json.loads((client / ".deployments/version-123.json").read_text())
            self.assertEqual(receipt["versionId"], "version-123")
            self.assertTrue(receipt["verified"])
            self.assertFalse(receipt["dirty"])
            self.assertEqual(receipt["tag"], "web-v1.2.3")
            verify.assert_called_once_with(receipt["url"])

    def test_changed_checkout_or_stale_metadata_prevents_upload(self):
        for changed in [True, False]:
            with self.subTest(changed_checkout=changed), tempfile.TemporaryDirectory() as temporary:
                client = Path(temporary)
                (client / ".sentry-artifacts").mkdir()
                (client / ".sentry-artifacts/release.json").write_text(
                    json.dumps({"release": "craftsky-web@wrong-build"}))
                commits = ["a" * 40, "b" * 40 if changed else "a" * 40]
                with patch.object(deploy, "CLIENT", client), \
                        patch.object(deploy, "release_commit", side_effect=commits), \
                        patch.object(deploy.subprocess, "run") as run, \
                        patch("sys.argv", ["web-client-deploy", "preview", "web-v1.2.3"]):
                    message = "release changed" if changed else "metadata does not match"
                    with self.assertRaisesRegex(ValueError, message):
                        deploy.main()
                    self.assertEqual(run.call_count, 2)
                    self.assertEqual(run.call_args_list[1].kwargs["env"]["SENTRY_RELEASE"],
                                     "craftsky-web@1.2.3+aaaaaaaaaaaa")
                    self.assertFalse(any("upload" in call.args[0] for call in run.call_args_list))

    def test_verify_rejects_untrusted_destinations_before_network(self):
        for origin in ["http://app.craftsky.social", "https://evil.example", "https://app.craftsky.social@evil.workers.dev", "https://app.craftsky.social/path", "https://app.craftsky.social?token=canary"]:
            with self.subTest(origin=origin), patch.object(deploy, "build_opener") as opener:
                with self.assertRaises(ValueError):
                    deploy.verify(origin)
                opener.assert_not_called()

    def test_redirects_are_not_followed(self):
        self.assertIsNone(deploy.NoRedirects().redirect_request(None, None, 302, "", {}, "https://evil.example"))

    @staticmethod
    def hosted_response(request, **kwargs):
        paths = {
            "/.well-known/assetlinks.json": "verified-links/.well-known/assetlinks.json",
            "/.well-known/apple-app-site-association": "verified-links/.well-known/apple-app-site-association",
            "/auth/complete": "verified-links/auth/complete.html",
            "/account-deletion/reauth-complete": "verified-links/account-deletion/reauth-complete.html",
        }
        path = deploy.urlsplit(request.full_url).path
        headers = {
            "X-Robots-Tag": "noindex", "Cache-Control": "no-store",
            "Referrer-Policy": "no-referrer",
            "Content-Type": "application/json" if path.startswith("/.well-known/") else "text/html; charset=utf-8",
        }
        if path in ("/", "/sign-in"):
            headers["Location"] = "/feed" if path == "/" else "/feed?auth=1"
            raise HTTPError(request.full_url, 303, "See Other", headers, io.BytesIO(b""))
        result = MagicMock()
        result.__enter__.return_value = result
        result.status = 200
        result.headers = headers
        result.read.return_value = (deploy.ROOT / paths[path]).read_bytes() if path in paths else b"<html/>"
        return result

    def test_hosted_verification_accepts_guest_redirects_and_checks_destinations_and_native_files(self):
        with patch.object(deploy, "build_opener") as opener:
            opener.return_value.open.side_effect = self.hosted_response
            deploy.verify("https://preview.example.workers.dev")
            requested = [call.args[0].full_url for call in opener.return_value.open.call_args_list]
            self.assertEqual(len(requested), 8)
            self.assertIn("https://preview.example.workers.dev/feed", requested)
            self.assertIn("https://preview.example.workers.dev/feed?auth=1", requested)

    def test_verification_rejects_foreign_or_unexpected_guest_redirects(self):
        for location in ["https://evil.example/feed", "//evil.example/feed", "/unexpected", "/feed?auth=1"]:
            with self.subTest(location=location), patch.object(deploy, "build_opener") as opener:
                opener.return_value.open.side_effect = HTTPError(
                    "https://preview.example.workers.dev/", 303, "See Other",
                    {"Location": location}, io.BytesIO(b""))
                with self.assertRaisesRegex(ValueError, "unexpected redirect"):
                    deploy.verify("https://preview.example.workers.dev")
                self.assertEqual(opener.return_value.open.call_count, 1)

    def test_guest_redirect_still_requires_privacy_headers(self):
        def response(request, **kwargs):
            try:
                return self.hosted_response(request, **kwargs)
            except HTTPError as error:
                error.headers.pop("Cache-Control")
                raise
        with patch.object(deploy, "build_opener") as opener:
            opener.return_value.open.side_effect = response
            with self.assertRaisesRegex(ValueError, "missing no-store"):
                deploy.verify("https://preview.example.workers.dev")

    def test_worker_config_has_no_live_domain(self):
        config = json.loads(deploy.CONFIG.read_text())
        self.assertFalse(config["workers_dev"])
        self.assertTrue(config["preview_urls"])
        self.assertFalse(config.get("routes"))
        self.assertFalse(config.get("route"))
        self.assertFalse(config["upload_source_maps"])


if __name__ == "__main__":
    unittest.main()
