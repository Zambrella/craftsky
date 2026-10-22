import json
from pathlib import Path
import unittest
import io
import contextlib
import tempfile
from unittest import mock
import service_status as status

from service_status import validate_document


class ContractTests(unittest.TestCase):
    # UT-007 / FR-010, FR-012 / AC-014, AC-016
    def test_schema_two_independent_controls(self):
        document = {
            'schemaVersion': 2, 'revision': 'D1',
            'maintenance': {'title': 'Break', 'description': 'Back soon'},
            'minimumVersions': {'ios': '1.8.0'},
            'message': {'revision': 'A', 'minVersion': '1.8.0',
                        'title': 'News', 'description': 'Hello',
                        'cta': {'text': 'Settings', 'destination': 'settings'}}}
        self.assertEqual(validate_document(json.dumps(document).encode()), document)
        with self.assertRaises(ValueError):
            validate_document(json.dumps({**document, 'message': None}).encode())

    # UT-002 / FR-002, FR-003, NFR-001, RULE-003 / AC-005, AC-019
    def test_literal_contract_corpus(self):
        corpus = json.loads((Path(__file__).resolve().parents[1] /
                             'service-status/contract-fixtures.json').read_text())
        for case in corpus:
            with self.subTest(case=case['name']):
                raw = json.dumps(case['document']).encode()
                if case['accepted']:
                    validate_document(raw)
                else:
                    with self.assertRaises(ValueError):
                        validate_document(raw)

    def test_literal_size_and_scalar_boundaries(self):
        base = {'schemaVersion': 2, 'revision': 'A',
                'maintenance': {'title': '🧶' * 160, 'description': 'x' * 2400}}
        validate_document(json.dumps(base, ensure_ascii=False).encode())
        for field, value in [('title', '🧶' * 161), ('description', 'x' * 2401)]:
            with self.subTest(field=field), self.assertRaises(ValueError):
                validate_document(json.dumps({**base, 'maintenance': {**base['maintenance'], field: value}}, ensure_ascii=False).encode())
        raw = b'{"schemaVersion":2,"revision":"A"}'
        validate_document(raw.ljust(16384))
        for value in [raw.ljust(16385), b'[]', b'null', b'<html>', b'{', b'\xff',
                      b'{"schemaVersion":2,"revision":"A","unknown":NaN}']:
            with self.subTest(value=value[:10]), self.assertRaises(ValueError):
                validate_document(value)


class FakeStore:
    def __init__(self):
        self.documents = []
        self.target = {'account': 'test-account', 'bucket': 'craftsky-service-status-preview',
                'environment': 'preview', 'url': 'https://preview-status.craftsky.social/app.json'}
    def put(self, document):
        self.documents.append(document)


class VersionTests(unittest.TestCase):
    # UT-001 / RULE-002 / AC-003, AC-010, AC-016
    def test_precedence_and_reversed_range(self):
        ordered = ['1.8.0-alpha.2', '1.8.0-alpha.10', '1.8.0-beta',
                   '1.8.0', '1.9.0', '1.10.0', '999999999999999.0.0']
        for a, b in zip(ordered, ordered[1:]):
            self.assertLess(status.compare_versions(a, b), 0)
        self.assertEqual(status.compare_versions('1.8.0+13', '1.8.0+99'), 0)
        with self.assertRaises(ValueError):
            validate_document(json.dumps({'schemaVersion': 2, 'revision': 'D',
                'message': {'revision': 'A', 'minVersion': '1.9.0', 'maxVersion': '1.8.0',
                            'title': 'News', 'description': 'Body'}}).encode())


class SchemaGateTests(unittest.TestCase):
    # UT-009 / FR-014 / AC-023
    def test_offline_formal_schema_and_fail_closed_final_payload(self):
        from jsonschema import Draft202012Validator
        schema_path = status.ROOT / 'service-status/app.schema.json'
        self.assertTrue(schema_path.is_file(), 'Versioned formal schema is missing')
        Draft202012Validator.check_schema(json.loads(schema_path.read_text()))
        broken_schema = json.loads(schema_path.read_text())
        broken_schema['$defs']['message']['properties']['title'] = {'$ref': '#/$defs/missing'}
        with tempfile.TemporaryDirectory() as temp:
            broken = Path(temp) / 'schema.json'
            for content in [b'{}', b'{', json.dumps(broken_schema).encode(), json.dumps({'type': 'invalid'}).encode()]:
                broken.write_bytes(content)
                store = FakeStore()
                with mock.patch.object(status, 'SCHEMA_PATH', broken):
                    self.assertEqual(status.run_command(['clear', '--environment', 'preview'], store=store,
                        confirm=lambda _: 'publish', output=io.StringIO()), 1)
                self.assertEqual(store.documents, [])
        for broken in [Path('/nonexistent/status-schema.json')]:
            store = FakeStore()
            with mock.patch.object(status, 'SCHEMA_PATH', broken):
                self.assertEqual(status.run_command(['clear', '--environment', 'preview'],
                    store=store, confirm=lambda _: 'publish', output=io.StringIO()), 1)
            self.assertEqual(store.documents, [])
        store = FakeStore()
        with mock.patch.object(status.uuid, 'uuid4', return_value='invalid revision'):
            self.assertEqual(status.run_command(['clear', '--environment', 'preview'],
                store=store, confirm=lambda _: 'publish', output=io.StringIO()), 1)
        self.assertEqual(store.documents, [])

    def test_schema_structure_and_semantic_range_are_distinct_layers(self):
        from jsonschema import Draft202012Validator, FormatChecker, ValidationError
        validator = Draft202012Validator(json.loads(status.SCHEMA_PATH.read_text()),
            format_checker=FormatChecker(formats=['uri', 'date-time']))
        validator.validate(json.loads((status.ROOT / 'service-status/example.json').read_text()))
        base = {'schemaVersion': 2, 'revision': 'D'}
        message = {'revision': 'A', 'title': 'News', 'description': 'Body', 'minVersion': '1.9.0', 'maxVersion': '1.8.0'}
        validator.validate({**base, 'message': message})
        with self.assertRaises(ValueError): validate_document(json.dumps({**base, 'message': message}).encode())
        for document in [{**base, 'maintenance': None}, {**base, 'minimumVersions': {'ios': 42}},
                         {**base, 'message': {**message, 'title': None}},
                         {**base, 'message': {**message, 'cta': {'text': 'Go', 'url': 'https://example.test', 'destination': 'feed'}}}]:
            with self.assertRaises(ValidationError): validator.validate(document)

    def test_validator_failure_prevents_upload_without_dependency_prose(self):
        for target in ['jsonschema.Draft202012Validator', 'jsonschema.FormatChecker']:
            store = FakeStore(); output = io.StringIO()
            with mock.patch(target, side_effect=RuntimeError('validator-secret-canary')):
                self.assertEqual(status.run_command(['clear', '--environment', 'preview'], store=store,
                    confirm=lambda _: 'publish', output=output), 1)
            self.assertEqual(store.documents, [])
            self.assertNotIn('validator-secret-canary', output.getvalue())


class WorkflowTests(unittest.TestCase):
    # AT-008 / BR-002, FR-006, FR-012 / AC-002, AC-009, AC-015.
    def test_validate_publish_clear_are_independent_and_generate_revisions(self):
        store = FakeStore()
        output = io.StringIO()
        draft = {'schemaVersion': 2, 'revision': 'copied-A',
                 'message': {'revision': 'campaign-A', 'minVersion': '1.8.0', 'title': 'Notice', 'description': 'Public information'},
                 'minimumVersions': {'ios': '1.8.0'},
                 'maintenance': {'title': 'Break', 'description': 'Back soon'}}
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / 'notice.json'
            original = json.dumps(draft).encode()
            path.write_bytes(original)
            with mock.patch('subprocess.run', side_effect=AssertionError('external release unavailable')):
                self.assertEqual(status.run_command(['validate', str(path)], output=output), 0)
                without_maintenance = Path(temp) / 'no-maintenance.json'
                without_maintenance.write_text(json.dumps({key: value for key, value in draft.items() if key != 'maintenance'}))
                for command in [['publish', str(path)], ['publish', str(without_maintenance)], ['clear']]:
                    self.assertEqual(status.run_command(command + ['--environment', 'preview'], store=store,
                        verifier=lambda target, document: True, confirm=lambda prompt: 'publish', output=output), 0)
            self.assertEqual(path.read_bytes(), original)
        self.assertEqual(len(store.documents), 3)
        revisions = [document['revision'] for document in store.documents]
        self.assertEqual(len(set(revisions)), 3)
        self.assertNotIn('copied-A', revisions)
        self.assertEqual(set(store.documents[-1]), {'schemaVersion', 'revision'})
        self.assertEqual([d['message']['revision'] for d in store.documents[:2]], ['campaign-A', 'campaign-A'])
        self.assertNotIn('maintenance', store.documents[1])
        self.assertEqual(store.documents[1]['minimumVersions'], draft['minimumVersions'])
        self.assertIn('PUBLIC', output.getvalue())
        self.assertIn('Verified', output.getvalue())


class SafetyTests(unittest.TestCase):
    # IT-003 / FR-003, FR-006, FR-012, RULE-002 / AC-005, AC-009, AC-015, AC-019.
    def test_invalid_declined_and_eof_never_upload(self):
        for raw, answer in [(b'<html>secret-canary', 'publish'),
                            (b'{"schemaVersion":2,"revision":"A"}', ''),
                            (b'{"schemaVersion":2,"revision":"A"}', EOFError())]:
            store = FakeStore(); output = io.StringIO()
            def confirm(prompt):
                if isinstance(answer, Exception): raise answer
                return answer
            with tempfile.TemporaryDirectory() as temp:
                path = Path(temp) / 'input.json'; path.write_bytes(raw)
                self.assertEqual(status.run_command(['publish', str(path), '--environment', 'preview'],
                    store=store, confirm=confirm, output=output), 1)
            self.assertEqual(store.documents, [])
            self.assertNotIn('secret-canary', output.getvalue())

    def test_environment_is_explicit_and_invalid_arguments_are_safe(self):
        for args in [['clear'], ['clear', '--environment', 'credential-canary']]:
            output = io.StringIO()
            with contextlib.redirect_stderr(output), self.assertRaises(SystemExit):
                status.run_command(args, output=output)
            self.assertNotIn('credential-canary', output.getvalue())

    def test_upload_replaces_only_the_selected_status_object(self):
        account = '0123456789abcdef0123456789abcdef'
        uploaded = []
        def runner(command, **kwargs):
            path = Path(command[command.index('--file') + 1])
            uploaded.append((path, validate_document(path.read_bytes())))
            self.assertIn('craftsky-service-status-preview/app.json', command)
            self.assertIn('--remote', command)
            self.assertNotIn('deploy', command)
            self.assertEqual(kwargs['env']['CLOUDFLARE_ACCOUNT_ID'], account)
            self.assertIs(kwargs['stdout'], status.subprocess.DEVNULL)
            self.assertIs(kwargs['stderr'], status.subprocess.DEVNULL)
            return mock.Mock(returncode=0)
        with mock.patch.object(status.subprocess, 'run', side_effect=runner) as run, \
             mock.patch.dict(status.os.environ, {'CLOUDFLARE_ACCOUNT_ID': account}):
            store = status.WranglerStatusObjectStore('preview')
            store.put({'schemaVersion': 2, 'revision': 'A'})
        self.assertEqual(run.call_count, 1)
        self.assertEqual(uploaded[0][1]['revision'], 'A')
        self.assertFalse(uploaded[0][0].exists())

    def test_setup_and_upload_errors_have_safe_outputs(self):
        output = io.StringIO()
        with mock.patch.dict(status.os.environ, {'CLOUDFLARE_ACCOUNT_ID': 'credential-canary'}):
            self.assertEqual(status.run_command(['clear', '--environment', 'preview'], output=output), 1)
        self.assertNotIn('credential-canary', output.getvalue())
        self.assertIn('no upload attempted', output.getvalue())
        store = FakeStore(); store.put = mock.Mock(side_effect=RuntimeError('credential-canary'))
        output = io.StringIO()
        self.assertEqual(status.run_command(['clear', '--environment', 'preview'], store=store,
            confirm=lambda _: 'publish', output=output), 1)
        self.assertNotIn('credential-canary', output.getvalue())
        self.assertIn('live state may have changed', output.getvalue())
        store.put.assert_called_once()


class VerificationTests(unittest.TestCase):
    # IT-004 / FR-012, NFR-002 / AC-015, AC-016.
    def test_public_read_checks_body_headers_and_does_not_follow_redirects(self):
        document = {'schemaVersion': 2, 'revision': 'fresh-A'}
        headers = {'content-type': 'application/json; charset=utf-8', 'cache-control': 'no-store',
                   'access-control-allow-origin': '*', 'x-content-type-options': 'nosniff'}
        response = mock.Mock(status=200)
        response.getheader.side_effect = lambda key: headers.get(key.lower())
        response.read.return_value = json.dumps(document).encode()
        connection = mock.Mock(); connection.getresponse.return_value = response
        with mock.patch.object(status.http.client, 'HTTPSConnection', return_value=connection):
            self.assertEqual(status.read_public_once(status.TARGETS['preview']['url']), document)
            for changes in [{'content-type': 'text/html'}, {'cache-control': 'max-age=60'},
                            {'access-control-allow-origin': 'https://wrong.invalid'},
                            {'access-control-allow-credentials': 'true'}]:
                bad = {**headers, **changes}
                response.getheader.side_effect = lambda key: bad.get(key.lower())
                with self.assertRaises(ValueError): status.read_public_once(status.TARGETS['preview']['url'])
            response.getheader.side_effect = lambda key: headers.get(key.lower())
            response.status = 302
            with self.assertRaises(ValueError): status.read_public_once(status.TARGETS['preview']['url'])
            response.status = 200; response.read.return_value = b'x' * 16385
            with self.assertRaises(ValueError): status.read_public_once(status.TARGETS['preview']['url'])
        response.read.assert_called_with(16385)
        request = connection.request.call_args
        self.assertEqual(request.args[:2], ('GET', '/app.json'))
        self.assertEqual(request.kwargs['headers'], {'Accept': 'application/json', 'Cache-Control': 'no-store'})

    def test_public_verification_compares_the_whole_intended_document(self):
        document = {'schemaVersion': 2, 'revision': 'A'}
        target = status.TARGETS['preview']
        with mock.patch.object(status, 'read_public_once', return_value=document):
            self.assertTrue(status.verify_public(target, document))
        for received in [{**document, 'revision': 'B'}, {**document, 'maintenance': {'title': 'Break', 'description': 'Back soon'}},
                         {**document, 'title': 'unexpected'}]:
            with self.subTest(received=received), mock.patch.object(status, 'read_public_once', return_value=received):
                self.assertFalse(status.verify_public(target, document))

    def test_concurrent_newer_publication_remains_unverified_without_rewrite(self):
        store = FakeStore(); output = io.StringIO()
        self.assertEqual(status.run_command(['clear', '--environment', 'preview'], store=store,
            verifier=lambda target, document: False, confirm=lambda _: 'publish', output=output), 1)
        self.assertEqual(len(store.documents), 1)
        self.assertIn('Unverified', output.getvalue())
        self.assertNotIn('Verified', output.getvalue())
        self.assertIn('live state may have changed', output.getvalue())


class LocalWorkflowTests(unittest.TestCase):
    # IT-003 / FR-013, FR-014 / AC-017, AC-023
    def test_local_publish_validates_final_payload_and_preserves_campaign(self):
        import runpy
        module = runpy.run_path(str(status.ROOT / 'scripts/service_status_dev.py'))
        publish = module['publish']
        globals_ = publish.__globals__
        writes = []
        with tempfile.TemporaryDirectory() as temp:
            def upload(*arguments):
                self.assertIn('--local', arguments)
                raw = Path(arguments[arguments.index('--file') + 1]).read_bytes()
                writes.append(validate_document(raw))
            with mock.patch.dict(globals_, {'LOCAL': Path(temp), 'wrangler': upload}):
                draft = {'schemaVersion': 2, 'revision': 'D', 'message': {'revision': 'A', 'minVersion': '1.8.0', 'title': 'News', 'description': 'Body'}}
                with contextlib.redirect_stdout(io.StringIO()):
                    publish(draft)
                self.assertEqual(writes[0]['message']['revision'], 'A')
                with mock.patch.object(status.uuid, 'uuid4', return_value='bad revision'):
                    with self.assertRaises(ValueError): publish(draft)
        self.assertEqual(len(writes), 1)


class IndependenceTests(unittest.TestCase):
    # IT-005 / FR-001, FR-012 / AC-001, AC-015.
    def test_separate_resource_configuration_cannot_bundle_live_status(self):
        config = json.loads((status.ROOT / 'service-status/wrangler.json').read_text())
        landing = json.loads((status.ROOT / 'web/wrangler.json').read_text())
        self.assertNotEqual(config['name'], landing['name'])
        self.assertNotIn('assets', config)
        self.assertEqual(config['r2_buckets'][0]['binding'], 'STATUS_DOCUMENTS')
        self.assertEqual(config['r2_buckets'][0]['bucket_name'], status.TARGETS['production']['bucket'])
        preview = config['env']['preview']
        self.assertNotEqual(preview['name'], config['name'])
        self.assertEqual(preview['r2_buckets'][0]['bucket_name'], status.TARGETS['preview']['bucket'])
        self.assertNotEqual(preview['r2_buckets'][0]['bucket_name'], config['r2_buckets'][0]['bucket_name'])
        self.assertEqual(config['routes'][0]['pattern'], 'status.craftsky.social')
        self.assertEqual(preview['routes'][0]['pattern'], 'preview-status.craftsky.social')
        self.assertNotIn('r2_buckets', landing)
        self.assertFalse(config['workers_dev'])


if __name__ == '__main__':
    unittest.main()
