import copy
import importlib.machinery
import importlib.util
import io
import json
import unittest
from pathlib import Path
from unittest.mock import patch

path = Path(__file__).with_name('oauth-permissions-check')
loader = importlib.machinery.SourceFileLoader('oauth_permissions_check', str(path))
spec = importlib.util.spec_from_loader(loader.name, loader)
module = importlib.util.module_from_spec(spec)
loader.exec_module(module)


class PermissionVerificationTests(unittest.TestCase):
    def setUp(self):
        self.expected = json.loads(module.source('lexicon/social/craftsky/authFull.json', None))
        self.scopes = next(line.split('=', 1)[1] for line in module.source('appview/environments/prod.env.example', None).splitlines() if line.startswith('OAUTH_SCOPES='))
        self.record = {'value': {'$type': 'com.atproto.lexicon.schema', **self.expected}}
        self.metadata = {'client_id': 'https://appview.craftsky.social/oauth-client-metadata.json', 'scope': self.scopes}
        self.dns = {'Answer': [{'type': 16, 'data': '"did=' + module.DID + '"'}]}
        self.identity = {'id': module.DID, 'alsoKnownAs': ['at://craftsky.social'], 'service': [{'id': '#atproto_pds', 'serviceEndpoint': 'https://pds.example'}]}

    def verify(self, publication_only=False):
        def read(url):
            if 'plc.directory' in url:
                return copy.deepcopy(self.identity)
            if 'com.atproto.repo.getRecord' in url:
                return copy.deepcopy(self.record)
            return copy.deepcopy(self.metadata)
        with patch.object(module, 'read_json', side_effect=read), patch.object(module.urllib.request, 'urlopen', side_effect=lambda *_args, **_kwargs: io.BytesIO(json.dumps(self.dns).encode())):
            module.check('https://appview.craftsky.social', publication_only=publication_only)

    def test_current_publication_and_metadata_pass(self):
        self.verify()

    def test_unpublished_or_changed_permission_set_blocks(self):
        self.record['value']['defs']['main']['permissions'][0]['collection'].append('social.craftsky.extra.record')
        with self.assertRaisesRegex(ValueError, 'Published permission set'):
            self.verify(True)

    def test_wrong_authority_blocks(self):
        self.dns['Answer'][0]['data'] = '"did=did:plc:other"'
        with self.assertRaisesRegex(ValueError, 'DNS authority'):
            self.verify(True)

    def test_stale_production_scope_blocks_success(self):
        self.metadata['scope'] = 'atproto transition:generic'
        with self.assertRaisesRegex(ValueError, 'scopes'):
            self.verify()

    def test_missing_scope_and_duplicate_scope_block(self):
        for scope in ['atproto', self.scopes + ' atproto']:
            self.metadata['scope'] = scope
            with self.assertRaisesRegex(ValueError, 'scopes'):
                self.verify()

    def test_wrong_client_id_blocks(self):
        self.metadata['client_id'] = 'https://appview.craftsky.social/oauth/client-metadata.json'
        with self.assertRaisesRegex(ValueError, 'client ID'):
            self.verify()

    def test_preflight_does_not_require_deployed_metadata(self):
        self.metadata = {}
        self.verify(True)

    def test_release_artifacts_are_read_from_exact_commit(self):
        with patch.object(module.subprocess, 'run') as run:
            run.return_value.stdout = 'tagged content'
            self.assertEqual(module.source('lexicon/social/craftsky/authFull.json', 'abc123'), 'tagged content')
            self.assertEqual(run.call_args.args[0], ['git', 'show', 'abc123:lexicon/social/craftsky/authFull.json'])


if __name__ == '__main__':
    unittest.main()
