"""Exercise worktree credential inheritance, secret parsing and redacted diagnostics."""
import contextlib
import importlib.machinery
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
loader = importlib.machinery.SourceFileLoader('config_tool', str(ROOT / 'scripts/config'))
spec = importlib.util.spec_from_loader(loader.name, loader)
config = importlib.util.module_from_spec(spec)
loader.exec_module(config)


class ConfigTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.build = tempfile.TemporaryDirectory()
        cls.binary = Path(cls.build.name) / 'configtool'
        subprocess.run(['go', 'build', '-o', str(cls.binary), './cmd/configtool'], cwd=ROOT / 'appview', check=True)

    @classmethod
    def tearDownClass(cls):
        cls.build.cleanup()

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.source = self.root / 'source'
        self.target = self.root / 'target'
        self.source.mkdir()
        self.target.mkdir()
        subprocess.run(['git', 'init', '-q', str(self.target)], check=True)
        (self.target / '.gitignore').write_text('.env.local\n.env.secrets.local\napp/config/*.env\n')
        self.addCleanup(patch.stopall)
        patch.object(config, 'primary', return_value=self.source).start()
        patch.object(config, 'helper', side_effect=self.helper).start()

    def helper(self, action, path, data=None):
        result = subprocess.run([str(self.binary), action, str(path)], capture_output=True, text=True,
                                input=json.dumps(data) if data is not None else None)
        if result.returncode:
            raise ValueError('configuration parse failed; values withheld')
        return json.loads(result.stdout)

    def initialize(self):
        with contextlib.redirect_stdout(io.StringIO()):
            config.init(self.target)

    def test_inherits_credentials_without_ports_production_or_overwrites(self):
        (self.source / '.env.local').write_text('BUNNY_STREAM_API_KEY=source-canary\nCRAFTSKY_APPVIEW_PORT=18080\nOAUTH_CLIENT_SECRET_KEY=production-canary\nDATABASE_URL=production-canary\nSENTRY_AUTH_TOKEN=deploy-canary\nE2E_PASSWORD=login-canary\n')
        (self.target / '.env.local').write_text('BUNNY_STREAM_API_KEY=destination-canary\nCRAFTSKY_APPVIEW_PORT=18999\n')
        (self.source / 'app/config').mkdir(parents=True)
        (self.source / 'app/config/production.env').write_text('PRIVATE=production-canary\n')
        (self.source / 'app/config/local.env').write_text('SENTRY_DSN=public-local\n')
        self.initialize()
        copied = self.helper('read', self.target / '.env.secrets.local')
        self.assertNotIn('BUNNY_STREAM_API_KEY', copied)
        self.assertNotIn('CRAFTSKY_APPVIEW_PORT', copied)
        self.assertNotIn('OAUTH_CLIENT_SECRET_KEY', copied)
        self.assertNotIn('DATABASE_URL', copied)
        self.assertNotIn('SENTRY_AUTH_TOKEN', copied)
        self.assertNotIn('E2E_PASSWORD', copied)
        self.assertFalse((self.target / 'app/config/production.env').exists())
        self.assertEqual(self.helper('read', self.target / 'app/config/local.env')['SENTRY_DSN'], 'public-local')
        self.assertEqual(len(copied['CHAT_OPERATION_HMAC_KEY']), 43)
        self.assertEqual((self.target / '.env.secrets.local').stat().st_mode & 0o777, 0o600)
        first = (self.target / '.env.secrets.local').read_bytes()
        self.initialize()
        self.assertEqual(first, (self.target / '.env.secrets.local').read_bytes())

    def test_quotes_dollars_backslashes_unicode_and_newlines_round_trip(self):
        values = {'BUNNY_STREAM_API_KEY': 'dollar$WORD \\ quoted\'" café\nnext', 'CHAT_OPERATION_HMAC_KEY': '000123'}
        path = self.source / '.env.local'
        config.write_missing(path, values)
        self.assertEqual(self.helper('read', path), values)
        self.initialize()
        copied = self.helper('read', self.target / '.env.secrets.local')
        self.assertEqual(copied['BUNNY_STREAM_API_KEY'], values['BUNNY_STREAM_API_KEY'])

    def test_old_base_uses_ignored_legacy_filename(self):
        (self.target / '.gitignore').write_text('.env.local\napp/config/*.env\n')
        self.initialize()
        self.assertTrue((self.target / '.env.local').exists())
        self.assertFalse((self.target / '.env.secrets.local').exists())

    def test_worktree_does_not_inherit_console_origin(self):
        (self.source / '.env.local').write_text('SAFETY_EMAIL_CONSOLE_ORIGIN=http://127.0.0.1:18080\n')
        self.initialize()
        copied = self.helper('read', self.target / '.env.secrets.local')
        self.assertNotIn('SAFETY_EMAIL_CONSOLE_ORIGIN', copied)

    def test_symlink_destination_is_rejected(self):
        outside = self.root / 'outside'
        outside.write_text('unchanged')
        (self.target / '.env.secrets.local').symlink_to(outside)
        with self.assertRaises(ValueError):
            self.initialize()
        self.assertEqual(outside.read_text(), 'unchanged')

    def test_parser_error_does_not_expose_canary(self):
        path = self.root / 'bad.env'
        path.write_text('SECRET="protected-canary\n')
        result = subprocess.run([str(self.binary), 'read', str(path)], capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn('protected-canary', result.stdout + result.stderr)

    def test_compose_accepts_serialized_special_characters(self):
        if not shutil.which('docker'):
            self.skipTest('Docker Compose not available')
        value = 'dollar$WORD and \'" café\nnext'
        config.write_missing(self.root / 'values.env', {'SECRET': value})
        compose = self.root / 'compose.yml'
        compose.write_text('services:\n  check:\n    image: alpine\n    env_file: values.env\n')
        result = subprocess.run(['docker', 'compose', '-f', str(compose), 'config', '--format', 'json'], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0)
        # Compose escapes dollar signs for its rendered configuration output.
        actual = json.loads(result.stdout)['services']['check']['environment']['SECRET']
        self.assertEqual(actual.replace('$$', '$'), value)

    def test_wrapper_exports_only_controls_and_parses_quoted_port(self):
        (self.root / '.env.local').write_text("CRAFTSKY_PUBLIC_APP_PORT='25444'\nBUNNY_STREAM_API_KEY=protected-canary\n")
        with patch.object(config, 'ROOT', self.root), patch.dict(os.environ, {}, clear=True), patch('sys.argv', ['config', 'shell']):
            output = io.StringIO()
            with contextlib.redirect_stdout(output):
                config.main()
            self.assertEqual(output.getvalue(), 'export CRAFTSKY_PUBLIC_APP_PORT=25444\n')
            self.assertNotIn('protected-canary', output.getvalue())

    def test_service_summary_merges_profiles_without_exposing_values(self):
        (self.root / 'appview/environments').mkdir(parents=True)
        (self.root / 'appview/internal/app').mkdir(parents=True)
        (self.root / 'appview/internal/app/config.go').write_text('boolEnv("LINK_PREVIEWS_ENABLED", env != EnvProd)\nboolEnv("PUSH_ENABLED", false)\nboolEnv("SENTRY_LOGS_ENABLED", false)\n')
        (self.root / 'appview/environments/dev.env').write_text('SENTRY_DSN=dsn-canary\nVIDEO_PROVIDER=bunny\n')
        (self.root / '.env.secrets.local').write_text('VIDEO_PROVIDER=bluesky\nREVENUECAT_API_KEY=secret-canary\nCHAT_OPERATION_HMAC_KEY=chat-canary\n')
        (self.root / '.env.local').write_text('VIDEO_ENABLED=false\nINSTAGRAM_META_ENABLED=false\nSENTRY_DSN=\n')
        with patch.object(config, 'ROOT', self.root), patch.dict(os.environ, {'INSTAGRAM_META_ENABLED': 'true', 'REVENUECAT_ENABLED': 'true'}, clear=True), patch('sys.argv', ['config', 'services', '--push-enabled']):
            output = io.StringIO()
            with contextlib.redirect_stdout(output):
                config.main()
        text = output.getvalue()
        for expected in ('Firebase push: enabled (ADC found)', 'Instagram Meta: enabled',
                         'RevenueCat: disabled', 'Sentry: disabled', 'Brevo safety emails: disabled',
                         'Bunny Stream video: disabled (provider not selected)', 'Bluesky video: disabled',
                         'Bunny CDN purge: not configured', 'Bluesky chat: configured'):
            self.assertIn(expected, text)
        self.assertNotIn('canary', text)
        self.assertIn('LINK_PREVIEWS_ENABLED: enabled', text)
        self.assertIn('PUSH_ENABLED: enabled', text)
        self.assertIn('SENTRY_LOGS_ENABLED: disabled (Sentry DSN not configured)', text)

    def test_service_summary_defaults_and_invalid_values_are_redacted(self):
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            config.service_summary({})
            config.service_summary({'INSTAGRAM_META_ENABLED': 'private-canary', 'VIDEO_PROVIDER': 'private-canary'})
        text = output.getvalue()
        self.assertIn('Bunny Stream video: enabled', text)
        self.assertIn('Firebase push: disabled (ADC not found)', text)
        self.assertIn('Instagram Meta: invalid configuration', text)
        self.assertIn('Bunny Stream video: invalid configuration', text)
        self.assertNotIn('private-canary', text)

    def test_feature_flags_follow_loader_defaults_and_redact_invalid_values(self):
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            config.service_summary({'VIDEO_ENABLED': 'false', 'LINK_PREVIEWS_ENABLED': 'private-canary',
                                    'INSTAGRAM_META_REPLIES_ENABLED': 'true'})
        text = output.getvalue()
        self.assertIn('VIDEO_ENABLED: disabled', text)
        self.assertIn('LINK_PREVIEWS_ENABLED: invalid configuration', text)
        self.assertIn('INSTAGRAM_META_REPLIES_ENABLED: enabled', text)
        self.assertIn('REVENUECAT_WEBHOOK_REQUIRE_SIGNATURE: enabled', text)
        self.assertIn('APPVIEW_UNSAFE_LOG_RESPONSE_BODIES: disabled', text)
        self.assertNotIn('private-canary', text)

    def test_refuses_to_copy_into_unignored_paths(self):
        (self.target / '.gitignore').write_text('')
        with self.assertRaises(ValueError):
            self.initialize()
        self.assertFalse((self.target / '.env.local').exists())
        self.assertFalse((self.target / '.env.secrets.local').exists())

    def test_fresh_backend_examples_initialize_and_validate(self):
        for name in ('.env.local', '.env.secrets.local'):
            shutil.copyfile(ROOT / (name + '.example'), self.target / name)
        profile = self.target / 'appview/environments'
        profile.mkdir(parents=True)
        shutil.copyfile(ROOT / 'appview/environments/dev.env', profile / 'dev.env')
        self.initialize()
        local = self.helper('read', self.target / '.env.local')
        secret = self.helper('read', self.target / '.env.secrets.local')
        for key in ('CHAT_OPERATION_HMAC_KEY', 'OAUTH_HANDOFF_RECEIPT_KEY', 'INSTAGRAM_DATA_HMAC_KEY'):
            self.assertTrue(secret[key])
            self.assertNotIn(key, local)
        for key in ('BUNNY_STREAM_API_KEY', 'BUNNY_CDN_PURGE_API_KEY', 'BREVO_API_KEY',
                    'INSTAGRAM_META_ACCESS_TOKEN', 'REVENUECAT_API_KEY'):
            self.assertEqual(secret[key], '')
            self.assertNotIn(key, local)
        with patch.dict(os.environ, {}, clear=True):
            self.assertTrue(self.helper('validate', self.target)['valid'])
