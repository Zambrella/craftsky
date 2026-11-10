"""Verify local launcher routing without starting Docker or contacting a backend."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

SOURCE = Path(__file__).with_name('public-app-dev')


class PublicAppDevTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / 'scripts').mkdir()
        (self.root / 'web-client').mkdir()
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        shutil.copy2(SOURCE, self.root / 'scripts/public-app-dev')
        compose = self.root / 'scripts/compose-dev'
        compose.write_text('''#!/usr/bin/env python3
import json, os, sys
with open(os.environ['LAUNCH_LOG'], 'a') as f:
 f.write(json.dumps({'command':'compose','args':sys.argv[1:]})+'\\n')
if os.environ.get('FIXTURE_PORT_FAILURE'): sys.exit(1)
if sys.argv[1]=='port': print(os.environ.get('FIXTURE_ADDRESS','127.0.0.1:18400'))
''')
        config = self.root / 'scripts/config'
        config.write_text('#!/bin/sh\nprintf \'25444\\n\'\n')
        config.chmod(0o755)
        npm = self.bin / 'npm'
        npm.write_text('''#!/usr/bin/env python3
import json, os, sys
with open(os.environ['LAUNCH_LOG'], 'a') as f:
 f.write(json.dumps({'command':'npm','args':sys.argv[1:],'cwd':os.getcwd(),'origin':os.environ.get('APPVIEW_ORIGIN'),'canary':os.environ.get('PRIVATE_CANARY')})+'\\n')
''')
        compose.chmod(0o755)
        npm.chmod(0o755)
        self.log = self.root / 'calls.jsonl'
        self.env = {**os.environ, 'PATH': str(self.bin) + os.pathsep + os.environ['PATH'],
                    'LAUNCH_LOG': str(self.log), 'APPVIEW_ORIGIN': 'https://production.invalid'}
        self.env.pop('CRAFTSKY_PUBLIC_APP_PORT', None)
        self.env.pop('PRIVATE_CANARY', None)

    def run_launcher(self, *args):
        result = subprocess.run([str(self.root / 'scripts/public-app-dev'), *args],
                                cwd='/', env=self.env, capture_output=True, text=True)
        calls = [json.loads(line) for line in self.log.read_text().splitlines()]
        self.assertTrue(all(call['args'] == ['port', 'appview', '8080']
                            for call in calls if call['command'] == 'compose'))
        return result, calls

    def test_selects_this_worktrees_backend_and_explicit_port_without_sourcing_secrets(self):
        (self.root / '.env.local').write_text('CRAFTSKY_PUBLIC_APP_PORT=25444\nPRIVATE_CANARY=protected\n')
        result, calls = self.run_launcher('--open')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(calls[0]['args'], ['port', 'appview', '8080'])
        self.assertEqual(calls[1]['args'], ['ci'])
        self.assertEqual(calls[-1]['args'], ['run', 'dev', '--', '--port', '25444', '--strictPort', '--open'])
        self.assertEqual(calls[-1]['origin'], 'http://127.0.0.1:18400')
        self.assertEqual(calls[-1]['cwd'], str((self.root / 'web-client').resolve()))
        self.assertIsNone(calls[-1]['canary'])

    def test_uses_stable_checkout_port_and_reuses_installed_dependencies(self):
        (self.root / 'web-client/node_modules').mkdir()
        result, calls = self.run_launcher()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(calls), 2)
        checksum = subprocess.check_output(['cksum'], input=str(self.root).encode()).split()[0]
        self.assertEqual(calls[-1]['args'][4], str(25000 + int(checksum) % 1000))

    def test_normalizes_all_interfaces_and_prefers_shell_override(self):
        self.env.update(FIXTURE_ADDRESS='0.0.0.0:19001', CRAFTSKY_PUBLIC_APP_PORT='25555')
        (self.root / '.env.local').write_text('CRAFTSKY_PUBLIC_APP_PORT=25444\n')
        result, calls = self.run_launcher()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(calls[-1]['origin'], 'http://127.0.0.1:19001')
        self.assertIn('25555', calls[-1]['args'])

    def test_missing_backend_requests_separate_start_without_launching_frontend(self):
        for overrides in [{'FIXTURE_PORT_FAILURE':'1'}, {'FIXTURE_ADDRESS':''}]:
            with self.subTest(overrides=overrides):
                self.log.unlink(missing_ok=True)
                self.env.update(overrides)
                result, calls = self.run_launcher()
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('just dev-d first', result.stderr)
                self.assertFalse(any(call['command']=='npm' for call in calls))
                for key in overrides:
                    self.env.pop(key)

    def test_rejects_nonlocal_backend_and_invalid_web_port(self):
        for overrides in [{'FIXTURE_ADDRESS':'production.invalid:8080'}, {'CRAFTSKY_PUBLIC_APP_PORT':'70000'}]:
            with self.subTest(overrides=overrides):
                self.log.unlink(missing_ok=True)
                self.env.update(overrides)
                result, calls = self.run_launcher()
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(call['command']=='npm' for call in calls))
                for key in overrides:
                    self.env.pop(key)


if __name__ == '__main__':
    unittest.main()
