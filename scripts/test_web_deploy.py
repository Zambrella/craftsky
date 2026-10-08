import importlib.machinery
import importlib.util
import io
import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from cloudflare_pages_build import PUBLIC_FILES, ROUTES, build_site

loader = importlib.machinery.SourceFileLoader("web_deploy", str(Path(__file__).with_name("web-deploy")))
spec = importlib.util.spec_from_loader(loader.name, loader)
deploy = importlib.util.module_from_spec(spec)
loader.exec_module(deploy)


class WebDeployTests(unittest.TestCase):
    def test_preview_alias_is_safe_for_worker_dns_names(self):
        self.assertEqual(deploy.preview_alias("main", "craftsky-landing"), "preview-main")
        self.assertEqual(deploy.preview_alias("Feature/Some Work", "craftsky-landing"), "preview-feature-some-work")
        self.assertLessEqual(len(deploy.preview_alias("x" * 100, "craftsky-landing")) + len("-craftsky-landing"), 63)

    def test_production_refuses_dirty_or_unpushed_checkout(self):
        with mock.patch.object(deploy, "git", return_value="modified"), mock.patch.object(deploy.subprocess, "run") as run:
            with self.assertRaisesRegex(ValueError, "clean working tree"):
                deploy.require_production_checkout("main")
            run.assert_not_called()
        with mock.patch.object(deploy, "git", side_effect=["", "feature"]), mock.patch.object(deploy.subprocess, "run") as run:
            with self.assertRaisesRegex(ValueError, "checking out main"):
                deploy.require_production_checkout("main")
            run.assert_not_called()
        with mock.patch.object(deploy, "git", side_effect=["", "main", "local", "remote"]), mock.patch.object(deploy.subprocess, "run"):
            with self.assertRaisesRegex(ValueError, "pushed origin/main"):
                deploy.require_production_checkout("main")

    def test_verification_checks_clean_policy_routes_and_rejects_stale_content(self):
        with tempfile.TemporaryDirectory() as temporary:
            web = Path(temporary)
            for name in PUBLIC_FILES:
                (web / name).write_text(name)
            output = build_site(web, "commit", False)
            requested = []

            def respond(request, timeout):
                self.assertEqual(request.get_header("User-agent"), "CraftSky-Website-Deploy/1.0")
                route = request.full_url.removeprefix("https://test.workers.dev")
                requested.append(route)
                name = ROUTES.get(route, route.lstrip("/"))
                return io.BytesIO((output / name).read_bytes())

            with mock.patch.object(deploy, "urlopen", side_effect=respond):
                deploy.verify_once("https://test.workers.dev", output)
            self.assertIn("/privacy", requested)
            self.assertIn("/terms", requested)
            self.assertIn("/styles.css", requested)
            with mock.patch.object(deploy, "urlopen", return_value=io.BytesIO(b"stale deployment")):
                with self.assertRaisesRegex(ValueError, "content differs"):
                    deploy.verify_once("https://test.workers.dev", output)

    def test_preview_upload_does_not_promote_live_traffic(self):
        self.check_upload("preview", True)

    def test_production_upload_verifies_deployment_and_public_origin(self):
        self.check_upload("production", False)

    def test_failed_version_verification_does_not_promote_production(self):
        self.check_upload("production", False, verification_fails=True)

    def test_disabled_version_urls_do_not_promote_production(self):
        self.check_upload("production", False, version_urls=False)

    def check_upload(self, environment, dirty, inherited_account="other-account", verification_fails=False, version_urls=True):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            web = root / "web"
            web.mkdir()
            for name in PUBLIC_FILES:
                (web / name).write_text(name)
            (web / "deploy.json").write_text(json.dumps({
                "productionBranch": "main",
                "publicOrigin": "https://craftsky.social",
            }))
            (web / "wrangler.json").write_text(json.dumps({"account_id": "test-account", "name": "craftsky-landing"}))
            wrangler = web / "node_modules" / ".bin" / "wrangler"
            wrangler.parent.mkdir(parents=True)
            wrangler.touch()
            calls = []

            def run(command, **kwargs):
                calls.append(command)
                if command == ["just", "web-check"]:
                    build_site(web, "commit", dirty)
                elif "upload" in command:
                    self.assertEqual(kwargs["env"]["CLOUDFLARE_ACCOUNT_ID"], "test-account")
                    Path(kwargs["env"]["WRANGLER_OUTPUT_FILE_PATH"]).write_text(json.dumps({
                        "type": "version-upload", "worker_name": "craftsky-landing",
                        "version_id": "123", "preview_url": "https://123-craftsky-landing.test.workers.dev" if version_urls else None,
                    }) + "\n")
                elif "deploy" in command:
                    self.assertEqual(kwargs["env"]["CLOUDFLARE_ACCOUNT_ID"], "test-account")
                    verify.assert_called_once_with("https://123-craftsky-landing.test.workers.dev", web / "dist")
                    with Path(kwargs["env"]["WRANGLER_OUTPUT_FILE_PATH"]).open("a") as records:
                        records.write(json.dumps({"type": "version-deploy", "worker_name": "craftsky-landing",
                                                 "deployment_id": "deployment-123", "version_traffic": {}}) + "\n")

            def git(*args):
                return {("rev-parse", "HEAD"): "commit", ("branch", "--show-current"): "feature",
                        ("log", "-1", "--format=%s"): "Website update"}[args]

            def list_projects(command, **kwargs):
                self.assertEqual(kwargs["env"]["CLOUDFLARE_ACCOUNT_ID"], "test-account")
                if "deployments" in command:
                    return '[{"id":"deployment-123","versions":[{"version_id":"123","percentage":100}]}]'
                return '[{"id":"existing-version"}]' 

            with mock.patch.object(deploy, "ROOT", root), mock.patch.object(deploy, "git", side_effect=git), \
                    mock.patch.object(deploy.subprocess, "check_output", side_effect=list_projects), \
                    mock.patch.object(deploy.subprocess, "run", side_effect=run), \
                    mock.patch.object(deploy, "require_production_checkout") as guard, \
                    mock.patch.object(deploy, "verify", side_effect=ValueError("verification failed") if verification_fails else None) as verify, \
                    mock.patch.dict(os.environ, {"CLOUDFLARE_ACCOUNT_ID": inherited_account} if inherited_account else {}, clear=True), \
                    mock.patch("sys.argv", ["web-deploy", environment]):
                if verification_fails or not version_urls:
                    with self.assertRaises(ValueError):
                        deploy.main()
                else:
                    self.assertEqual(deploy.main(), 0)
            upload = next(command for command in calls if "upload" in command)
            self.assertEqual(upload[upload.index("--name") + 1], "craftsky-landing")
            self.assertEqual(upload[upload.index("--preview-alias") + 1], "preview-feature")
            self.assertIn("commit-dirty" if dirty else "commit", upload)
            origins = ["https://123-craftsky-landing.test.workers.dev"]
            promotions = [command for command in calls if "deploy" in command]
            failed = verification_fails or not version_urls
            if environment == "production" and not failed:
                origins.append("https://craftsky.social")
                self.assertEqual(guard.call_count, 2)
                self.assertEqual(len(promotions), 1)
                self.assertIn("123@100", promotions[0])
            else:
                if environment == "preview":
                    guard.assert_not_called()
                self.assertEqual(promotions, [])
            if not version_urls:
                origins = []
            self.assertEqual(verify.call_args_list, [mock.call(origin, web / "dist") for origin in origins])
            receipt = json.loads((web / ".deployments" / "123.json").read_text())
            self.assertEqual(receipt["commit"], "commit")
            self.assertEqual(receipt["verifiedOrigins"], [] if failed else origins)

    def test_modified_or_extra_artifact_files_prevent_upload(self):
        with tempfile.TemporaryDirectory() as temporary:
            web = Path(temporary)
            for name in PUBLIC_FILES:
                (web / name).write_text(name)
            output = build_site(web, "commit", False)
            manifest = json.loads((output / "site-release.json").read_text())
            (output / ".env").write_text("private canary")
            with self.assertRaisesRegex(ValueError, "file list changed"):
                deploy.require_unchanged_artifact(output, manifest)
            (output / ".env").unlink()
            (output / "privacy.html").write_text("changed after checks")
            with self.assertRaisesRegex(ValueError, "artifact changed"):
                deploy.require_unchanged_artifact(output, manifest)

    def test_configured_account_is_used_without_environment_setup(self):
        self.check_upload("preview", True, inherited_account=None)


if __name__ == "__main__":
    unittest.main()
