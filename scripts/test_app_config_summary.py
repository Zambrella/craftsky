"""Verify effective Flutter flags and credential-free launcher diagnostics."""
import contextlib
import io
from pathlib import Path
import runpy
import os
import subprocess
import tempfile
import unittest

summary = runpy.run_path(str(Path(__file__).with_name('app-config-summary')))['summarize']


class AppConfigSummaryTests(unittest.TestCase):
    def test_app_launcher_prints_mobile_summary_before_backend_failure(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'scripts').mkdir()
            (root / 'app/config').mkdir(parents=True)
            (root / 'justfile').write_text(Path(__file__).resolve().parents[1].joinpath('justfile').read_text())
            for name in ('development', 'development-android'):
                (root / f'app/config/.env.{name}.local').write_text('')
            config = root / 'scripts/config'
            config.write_text('#!/bin/sh\nexit 0\n')
            config.chmod(0o755)
            (root / 'scripts/app-config-summary').write_text("print('Mobile services and feature flags')")
            compose = root / 'scripts/compose-dev'
            compose.write_text('#!/bin/sh\necho "no port 8080/tcp" >&2\nexit 1\n')
            compose.chmod(0o755)
            result = subprocess.run(['just', 'app-run'], cwd=root, capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('Mobile services and feature flags', result.stdout)
            self.assertIn('AppView is unavailable', result.stderr)
            self.assertNotIn('External services (backend', result.stdout + result.stderr)

    def test_compose_prints_backend_summary_only_on_startup(self):
        source = Path(__file__).resolve().parents[1]
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'scripts').mkdir()
            (root / 'scripts/compose-dev').write_text((source / 'scripts/compose-dev').read_text())
            for name, content in {
                'scripts/config': '#!/bin/sh\nif [ "$1" = services ]; then echo "Backend summary fixture"; fi\n',
                'git': '#!/bin/sh\ncase "$*" in *--show-toplevel*) echo "$FIXTURE_ROOT";; *) echo "$FIXTURE_ROOT/.git";; esac\n',
                'docker': '#!/bin/sh\necho "docker fixture"\n',
            }.items():
                path = root / name
                path.write_text(content)
                path.chmod(0o755)
            env = {**os.environ, 'PATH': str(root) + os.pathsep + os.environ['PATH'],
                   'FIXTURE_ROOT': str(root)}
            for args, expected in [(['port', 'appview', '8080'], False), (['up', '-d'], True)]:
                result = subprocess.run(['bash', str(root / 'scripts/compose-dev'), *args], env=env, capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual('Backend summary fixture' in result.stderr, expected)
                self.assertEqual('CraftSky stack:' in result.stderr, expected)

    def test_explicit_defines_override_files_and_never_print_keys(self):
        values = {'SUBSCRIPTIONS_ENABLED': 'false', 'SENTRY_DSN': 'dsn-canary',
                  'SENTRY_LOCAL_OPT_IN': 'true', 'REVENUECAT_TEST_STORE_PUBLIC_KEY': 'sdk-canary'}
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            summary(['--dart-define-from-file=config/.env.development.local',
                     '--dart-define=SUBSCRIPTIONS_ENABLED=true,REVENUECAT_USE_TEST_STORE=true',
                     '--dart-define', 'CRAFTSKY_DEV_OAUTH_SCHEME=true'], lambda _: values)
        text = output.getvalue()
        self.assertIn('RevenueCat ios: configured', text)
        self.assertIn('RevenueCat android: configured', text)
        self.assertIn('Sentry: configured (requires consent)', text)
        self.assertIn('SUBSCRIPTIONS_ENABLED: enabled', text)
        self.assertIn('CRAFTSKY_DEV_OAUTH_SCHEME (effective): enabled', text)
        self.assertNotIn('canary', text)

    def test_release_ignores_test_store_and_dev_oauth_and_respects_subscription_gate(self):
        values = {'SUBSCRIPTIONS_ENABLED': 'true', 'REVENUECAT_USE_TEST_STORE': 'true',
                  'REVENUECAT_TEST_STORE_PUBLIC_KEY': 'sdk-canary', 'CRAFTSKY_DEV_OAUTH_SCHEME': 'true',
                  'SENTRY_DSN': 'dsn-canary'}
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            summary(['--dart-define-from-file=config/.env.development.local', '--release'], lambda _: values)
        text = output.getvalue()
        self.assertIn('REVENUECAT_USE_TEST_STORE (effective): disabled', text)
        self.assertIn('CRAFTSKY_DEV_OAUTH_SCHEME (effective): disabled', text)
        self.assertIn('RevenueCat ios: disabled', text)
        self.assertIn('Sentry: disabled', text)
        self.assertNotIn('canary', text)
