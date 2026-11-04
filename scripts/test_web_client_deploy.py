"""Safety boundaries for the preview-only release command."""
import importlib.machinery
import importlib.util
import json
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
    def test_dirty_checkout_requires_explicit_experiment(self):
        with patch.object(deploy, "git", side_effect=["a" * 40, " M file"]), \
                patch.object(deploy.subprocess, "run") as run, \
                patch("sys.argv", ["web-client-deploy", "preview"]):
            with self.assertRaisesRegex(ValueError, "commit changes"):
                deploy.main()
            run.assert_not_called()

    def test_failed_checks_never_upload(self):
        with patch.object(deploy, "git", side_effect=["a" * 40, ""]), \
                patch.object(deploy.subprocess, "run", side_effect=[None, subprocess.CalledProcessError(1, "just")]) as run, \
                patch("sys.argv", ["web-client-deploy", "preview"]):
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
            (maps / "release.json").write_text(json.dumps({"release": "craftsky-web@test"}))
            commands = []

            def run(command, **kwargs):
                commands.append(command)
                if "upload" in command:
                    Path(kwargs["env"]["WRANGLER_OUTPUT_FILE_PATH"]).write_text(json.dumps({
                        "type": "version-upload", "worker_name": deploy.WORKER,
                        "version_id": "version-123", "preview_url": "https://version.example.workers.dev",
                    }) + "\n")

            with patch.object(deploy, "CLIENT", client), \
                    patch.object(deploy, "git", side_effect=["a" * 40, "", "a" * 40, ""]), \
                    patch.object(deploy.subprocess, "run", side_effect=run), \
                    patch.object(deploy, "verify") as verify, \
                    patch("sys.argv", ["web-client-deploy", "preview"]):
                self.assertEqual(deploy.main(), 0)
            upload = commands[-1]
            self.assertEqual(upload[2:4], ["versions", "upload"])
            self.assertIn("--preview-alias", upload)
            self.assertNotIn("deploy", upload)
            receipt = json.loads((client / ".deployments/version-123.json").read_text())
            self.assertEqual(receipt["versionId"], "version-123")
            self.assertTrue(receipt["verified"])
            self.assertFalse(receipt["dirty"])
            verify.assert_called_once_with(receipt["url"])

    def test_verify_rejects_untrusted_destinations_before_network(self):
        for origin in ["http://app.craftsky.social", "https://evil.example", "https://app.craftsky.social@evil.workers.dev", "https://app.craftsky.social/path", "https://app.craftsky.social?token=canary"]:
            with self.subTest(origin=origin), patch.object(deploy, "build_opener") as opener:
                with self.assertRaises(ValueError):
                    deploy.verify(origin)
                opener.assert_not_called()

    def test_redirects_are_not_followed(self):
        self.assertIsNone(deploy.NoRedirects().redirect_request(None, None, 302, "", {}, "https://evil.example"))

    def test_hosted_verification_accepts_cloudflare_noindex_and_checks_all_native_files(self):
        paths = {
            "/.well-known/assetlinks.json": "verified-links/.well-known/assetlinks.json",
            "/.well-known/apple-app-site-association": "verified-links/.well-known/apple-app-site-association",
            "/auth/complete": "verified-links/auth/complete.html",
            "/account-deletion/reauth-complete": "verified-links/account-deletion/reauth-complete.html",
        }
        def response(request, **kwargs):
            path = deploy.urlsplit(request.full_url).path
            result = MagicMock()
            result.__enter__.return_value = result
            result.status = 200
            result.headers = {
                "X-Robots-Tag": "noindex", "Cache-Control": "no-store",
                "Referrer-Policy": "no-referrer",
                "Content-Type": "application/json" if path.startswith("/.well-known/") else "text/html; charset=utf-8",
            }
            result.read.return_value = (deploy.ROOT / paths[path]).read_bytes() if path in paths else b"<html/>"
            return result
        with patch.object(deploy, "build_opener") as opener:
            opener.return_value.open.side_effect = response
            deploy.verify("https://preview.example.workers.dev")
            self.assertEqual(opener.return_value.open.call_count, 6)

    def test_worker_config_has_no_live_domain(self):
        config = json.loads(deploy.CONFIG.read_text())
        self.assertFalse(config["workers_dev"])
        self.assertTrue(config["preview_urls"])
        self.assertFalse(config.get("routes"))
        self.assertFalse(config.get("route"))
        self.assertFalse(config["upload_source_maps"])


if __name__ == "__main__":
    unittest.main()
