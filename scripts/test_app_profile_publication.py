import contextlib
import copy
import io
import json
import re
import shutil
import subprocess
import unittest
import urllib.error
from unittest import mock

import app_profile_publication as profile


class ProfileContractTests(unittest.TestCase):
    def test_declares_the_actual_record_collections_and_membership_marker(self):
        source = profile.load_source()
        registered = set(re.findall(r'collection: "([^"]+)"',
                                   (profile.ROOT / 'appview/internal/app/deps.go').read_text()))
        scope_source = (profile.ROOT / 'appview/internal/auth/scopes.go').read_text()
        scopes = set(re.findall(r'"repo:([^"?]+)', scope_source))
        # CraftSky record grants are now bundled in the included permission set.
        self.assertIn('"include:social.craftsky.authFull"', scope_source)
        permission_set = json.loads(
            (profile.ROOT / 'lexicon/social/craftsky/authFull.json').read_text())
        for permission in permission_set['defs']['main']['permissions']:
            if permission['resource'] == 'repo':
                scopes.update(permission['collection'])
        self.assertEqual(set(source['lexicons']['consumes']), registered)
        self.assertEqual(set(source['lexicons']['produces']), scopes)
        for records in source['lexicons'].values():
            self.assertEqual(len(records), len(set(records)))
        self.assertEqual(source['accountIndicators'], [
            {'collection': 'social.craftsky.actor.profile', 'rkey': 'self'}])

    def test_profile_declares_official_store_links_without_changing_launch_status(self):
        source = profile.load_source()
        self.assertEqual(source['status'], 'community.lexicon.app.defs#unreleased')
        self.assertEqual(source['links'][0]['uri'], 'https://craftsky.social')
        store_links = {link['role']: link['uri'] for link in source['links']
                       if link.get('role') in (
                           'community.lexicon.app.defs#linkRoleAppStore',
                           'community.lexicon.app.defs#linkRolePlayStore')}
        self.assertEqual(store_links, {
            'community.lexicon.app.defs#linkRoleAppStore':
                'https://apps.apple.com/app/id6812428632',
            'community.lexicon.app.defs#linkRolePlayStore':
                'https://play.google.com/store/apps/details?id=social.craftsky.app',
        })

    @unittest.skipUnless(shutil.which('goat'), 'goat is required for real pinned-schema validation')
    def test_real_validator_accepts_source_and_rejects_missing_required_metadata(self):
        source = profile.load_source()
        profile.validate(source)
        source.pop('name')
        with self.assertRaises(profile.PublicationError):
            profile.validate(source)


class PublicationTests(unittest.TestCase):
    def setUp(self):
        self.source = profile.load_source()
        self.existing = {'uri': profile.URI, 'cid': 'current-cid', 'value': copy.deepcopy(self.source)}
        self.pds = 'https://pds.example'

    def test_wrong_authenticated_account_never_uploads_or_writes(self):
        with mock.patch.object(profile, 'goat', return_value='DID: did:plc:another-owner\n') as goat:
            with self.assertRaisesRegex(profile.PublicationError, 'official'):
                profile.publish(self.source, self.pds, None)
        self.assertEqual(goat.call_args_list, [mock.call('account', 'check-auth')])

    def test_existing_equal_profile_is_verified_without_a_write_or_timestamp_churn(self):
        self.existing['value']['createdAt'] = '2026-01-01T00:00:00Z'
        self.existing['value']['updatedAt'] = '2026-02-01T00:00:00Z'
        self.existing['value']['futureExtension'] = {'retained': True}
        with mock.patch.object(profile, 'authenticated_owner'), \
                mock.patch.object(profile, 'current_record', return_value=self.existing), \
                mock.patch.object(profile, 'goat', side_effect=AssertionError('no write allowed')), \
                mock.patch.object(profile, 'validate') as validate:
            actual = profile.publish(self.source, self.pds, self.existing)
        self.assertEqual(actual, self.existing)
        validate.assert_not_called()

    def test_create_uploads_current_icon_and_uses_absent_record_compare_and_swap(self):
        calls = []
        def goat(*args, body=None):
            calls.append((args, body))
            if args[:2] == ('blob', 'upload'):
                return json.dumps({'blob': profile.icon_blob()})
            self.assertEqual(args, ('xrpc', 'procedure', '@pds', 'com.atproto.repo.putRecord', '-'))
            self.assertEqual(body['repo'], profile.DID)
            self.assertEqual(body['collection'], profile.COLLECTION)
            self.assertEqual(body['rkey'], 'self')
            # A reference PDS rejects unknown custom schemas with validate=true.
            # Leave its default enabled; our pinned validator runs before this.
            self.assertNotIn('validate', body)
            self.assertIn('swapRecord', body)
            self.assertIsNone(body['swapRecord'])
            self.assertEqual(body['record'], self.source)
            return json.dumps({'uri': profile.URI, 'cid': 'created-cid'})
        expected = {'uri': profile.URI, 'cid': 'created-cid', 'value': self.source}
        with mock.patch.object(profile, 'authenticated_owner'), \
                mock.patch.object(profile, 'validate') as validate, \
                mock.patch.object(profile, 'goat', side_effect=goat), \
                mock.patch.object(profile, 'current_record', return_value=expected):
            self.assertEqual(profile.publish(self.source, self.pds, None), expected)
        self.assertEqual(len(calls), 2)
        validate.assert_called_once_with(self.source)

    def test_update_preserves_extensions_and_creation_but_removes_withdrawn_metadata(self):
        self.existing['value']['createdAt'] = '2026-01-01T00:00:00Z'
        self.existing['value']['futureExtension'] = {'keep': 'original'}
        source = copy.deepcopy(self.source)
        source.pop('platforms')
        source['description'] = 'Updated public description.'
        def goat(*args, body=None):
            self.assertEqual(args[:2], ('xrpc', 'procedure'))
            self.assertEqual(body['swapRecord'], 'current-cid')
            self.assertEqual(body['record']['createdAt'], '2026-01-01T00:00:00Z')
            self.assertEqual(body['record']['futureExtension'], {'keep': 'original'})
            self.assertNotIn('platforms', body['record'])
            self.assertIn('updatedAt', body['record'])
            return json.dumps({'uri': profile.URI, 'cid': 'updated-cid'})
        with mock.patch.object(profile, 'authenticated_owner'), \
                mock.patch.object(profile, 'validate'), \
                mock.patch.object(profile, 'goat', side_effect=goat) as run, \
                mock.patch.object(profile, 'verify', return_value={'cid': 'updated-cid'}):
            profile.publish(source, self.pds, self.existing)
        self.assertEqual(run.call_count, 1)
        self.assertIn('platforms', self.existing['value'])

    def test_unexpected_blob_prevents_record_write(self):
        with mock.patch.object(profile, 'authenticated_owner'), \
                mock.patch.object(profile, 'validate'), \
                mock.patch.object(profile, 'goat', return_value=json.dumps({'blob': {'unexpected': True}})) as run:
            with self.assertRaisesRegex(profile.PublicationError, 'blob'):
                profile.publish(self.source, self.pds, None)
        self.assertEqual(run.call_count, 1)

    def test_concurrent_update_or_failed_write_is_not_retried_unconditionally(self):
        with mock.patch.object(profile, 'authenticated_owner'), \
                mock.patch.object(profile, 'validate'), \
                mock.patch.object(profile, 'goat', side_effect=profile.PublicationError('goat failed')) as run:
            changed = {**self.source, 'description': 'Updated.'}
            with self.assertRaisesRegex(profile.PublicationError, 'Run verify'):
                profile.publish(changed, self.pds, self.existing)
        self.assertEqual(run.call_count, 1)

    def test_public_readback_must_match_both_content_and_returned_cid(self):
        for value, cid in [({**self.source, 'name': 'Changed concurrently'}, 'created-cid'),
                           (self.source, 'different-cid')]:
            actual = {'uri': profile.URI, 'cid': cid, 'value': value}
            with self.subTest(cid=cid), mock.patch.object(profile, 'authenticated_owner'), \
                    mock.patch.object(profile, 'validate'), \
                    mock.patch.object(profile, 'goat', side_effect=[
                        json.dumps({'blob': profile.icon_blob()}),
                        json.dumps({'uri': profile.URI, 'cid': 'created-cid'})]), \
                    mock.patch.object(profile, 'current_record', return_value=actual):
                with self.assertRaises(profile.PublicationError):
                    profile.publish(self.source, self.pds, None)


class FailureTests(unittest.TestCase):
    def test_only_typed_record_not_found_allows_creation(self):
        for code, error in [(400, {'error': 'RecordNotFound'}),
                            (404, {'error': 'RecordNotFound'}),
                            (503, {'error': 'RecordNotFound'}),
                            (400, {'error': 'InvalidRequest'}),
                            (404, []), (404, 'credential-canary')]:
            failure = urllib.error.HTTPError('https://pds.example', code, 'hidden-canary', {},
                                            io.BytesIO(json.dumps(error).encode()))
            with self.subTest(code=code, error=error), mock.patch.object(profile, 'read', side_effect=failure):
                if code in (400, 404) and isinstance(error, dict) and error.get('error') == 'RecordNotFound':
                    self.assertIsNone(profile.current_record('https://pds.example'))
                else:
                    with self.assertRaises(profile.PublicationError):
                        profile.current_record('https://pds.example')

    def test_dependency_output_is_not_printed_and_original_cause_is_retained(self):
        original = subprocess.CalledProcessError(1, ['goat'], output='access-token-canary',
                                                stderr='refresh-token-canary')
        with mock.patch.object(profile.subprocess, 'run', side_effect=original):
            with self.assertRaises(profile.PublicationError) as caught:
                profile.goat('account', 'check-auth')
            self.assertIs(caught.exception.__cause__, original)
            output = io.StringIO()
            with contextlib.redirect_stderr(output):
                self.assertEqual(profile.main(['validate']), 1)
        self.assertIn('goat operation failed', output.getvalue())
        self.assertNotIn('token-canary', output.getvalue())


if __name__ == '__main__':
    unittest.main()
