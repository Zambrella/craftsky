"""Independent public status contract and operator tooling."""
import argparse
import datetime
import http.client
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
from urllib.parse import urlsplit
import uuid


MAX_BYTES = 16384
REVISION = re.compile(r'[A-Za-z0-9_-]{1,64}', re.ASCII)
TIMESTAMP = re.compile(r'\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?(?:Z|[+-]\d{2}:\d{2})', re.ASCII)


def _invalid_json_constant(value):
    raise ValueError('Status JSON')


def validate_document(raw):
    """Return recognized public fields; errors never include remote prose."""
    if len(raw) > MAX_BYTES:
        raise ValueError('Status size')
    try:
        value = json.loads(raw.decode('utf-8'), parse_constant=_invalid_json_constant)
    except (ValueError, UnicodeError, RecursionError):
        raise ValueError('Status JSON') from None
    if not isinstance(value, dict) or type(value.get('schemaVersion')) is not int or value['schemaVersion'] != 1:
        raise ValueError('Status schema')
    if value.get('mode') not in ('normal', 'announcement', 'maintenance'):
        raise ValueError('Status mode')
    revision = value.get('revision')
    if not isinstance(revision, str) or not REVISION.fullmatch(revision):
        raise ValueError('Status revision')
    result = {key: value[key] for key in ('schemaVersion', 'mode', 'revision')}
    for key, limit in [('title', 160), ('message', 2400)]:
        if key not in value and value['mode'] == 'normal':
            continue
        text = value.get(key)
        if not isinstance(text, str) or not text.strip() or len(text) > limit or any(0xD800 <= ord(c) <= 0xDFFF for c in text):
            raise ValueError('Status text')
        result[key] = text
    if 'estimatedRecoveryAt' in value:
        estimate = value['estimatedRecoveryAt']
        if not isinstance(estimate, str) or not TIMESTAMP.fullmatch(estimate):
            raise ValueError('Status estimate')
        try:
            # datetime rejects normalized dates, leap seconds and >23:59 offsets.
            datetime.datetime.fromisoformat(estimate.replace('Z', '+00:00'))
            if estimate[-6:-5] in ('+', '-') and (int(estimate[-5:-3]) > 23 or int(estimate[-2:]) > 59):
                raise ValueError('Status estimate')
        except ValueError:
            raise ValueError('Status estimate') from None
        result['estimatedRecoveryAt'] = estimate
    return result


ROOT = Path(__file__).resolve().parents[1]
TARGETS = {
    'production': {'environment': 'production', 'bucket': 'craftsky-service-status',
                   'url': 'https://status.craftsky.social/app.json'},
    'preview': {'environment': 'preview', 'bucket': 'craftsky-service-status-preview',
                'url': 'https://preview-status.craftsky.social/app.json'},
}


def encode_document(document):
    raw = json.dumps(document, ensure_ascii=False, separators=(',', ':')).encode('utf-8')
    validate_document(raw)
    return raw


class WranglerStatusObjectStore:
    def __init__(self, environment):
        self.target = dict(TARGETS[environment])
        self.account = os.environ.get('CLOUDFLARE_ACCOUNT_ID') or json.loads(
            (ROOT / 'web/wrangler.json').read_text())['account_id']
        if not re.fullmatch(r'[a-f0-9]{32}', self.account):
            raise ValueError('Invalid Cloudflare account')
        self.target['account'] = self.account

    def put(self, document):
        cli = ROOT / 'web/node_modules/wrangler/bin/wrangler.js'
        with tempfile.TemporaryDirectory(prefix='craftsky-status-') as temp:
            path = Path(temp) / 'app.json'
            path.write_bytes(encode_document(document))
            result = subprocess.run(['node', str(cli), 'r2', 'object', 'put',
                self.target['bucket'] + '/app.json', '--file', str(path),
                '--content-type', 'application/json', '--cache-control', 'no-store',
                '--remote', '--config', str(ROOT / 'service-status/wrangler.json')],
                cwd=ROOT, env={**os.environ, 'CLOUDFLARE_ACCOUNT_ID': self.account,
                               'WRANGLER_SEND_METRICS': 'false'},
                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                timeout=45, check=False)
            if result.returncode:
                raise ValueError('Cloudflare upload failed')


def read_public_once(url):
    if url not in {target['url'] for target in TARGETS.values()}:
        raise ValueError('Unknown public status origin')
    parsed = urlsplit(url)
    connection = http.client.HTTPSConnection(parsed.hostname, timeout=3)
    try:
        connection.request('GET', parsed.path, headers={'Accept': 'application/json', 'Cache-Control': 'no-store'})
        response = connection.getresponse()
        if (response.status != 200 or
            (response.getheader('content-type') or '').split(';')[0].strip().lower() != 'application/json' or
            'no-store' not in [part.strip().lower() for part in (response.getheader('cache-control') or '').split(',')] or
            response.getheader('access-control-allow-origin') != '*' or
            response.getheader('access-control-allow-credentials') is not None or
            response.getheader('set-cookie') is not None or
            response.getheader('x-content-type-options') != 'nosniff'):
            raise ValueError('Public status headers')
        return validate_document(response.read(MAX_BYTES + 1))
    finally:
        connection.close()


def verify_public(target, document):
    return read_public_once(target['url']) == document


class StatusArgumentParser(argparse.ArgumentParser):
    def error(self, message):
        # argparse otherwise quotes arbitrary invalid arguments verbatim.
        print('Invalid status command arguments; use --help.', file=sys.stderr)
        raise SystemExit(2)


def run_command(argv=None, *, store=None, verifier=None, confirm=input, output=None):
    output = output or sys.stdout
    parser = StatusArgumentParser(description='Independent public CraftSky status document')
    sub = parser.add_subparsers(dest='operation', required=True)
    validate = sub.add_parser('validate')
    validate.add_argument('file', type=Path)
    publish = sub.add_parser('publish')
    publish.add_argument('file', type=Path)
    clear = sub.add_parser('clear')
    for command in (publish, clear):
        command.add_argument('--environment', choices=TARGETS, required=True)
    args = parser.parse_args(argv)
    attempted = False
    try:
        if args.operation in ('validate', 'publish'):
            with args.file.open('rb') as source:
                document = validate_document(source.read(MAX_BYTES + 1))
        else:
            document = {'schemaVersion': 1, 'mode': 'normal', 'revision': 'draft'}
        if args.operation == 'validate':
            print('Valid status document', file=output)
            return 0
        document['revision'] = str(uuid.uuid4())
        encode_document(document)
        store = store or WranglerStatusObjectStore(args.environment)
        target = store.target
        print('PUBLIC status publication target:', file=output)
        print(json.dumps(target, sort_keys=True), file=output)
        print('Intended PUBLIC document:', file=output)
        print(json.dumps(document, ensure_ascii=False, indent=2), file=output)
        output.flush()
        try:
            answer = confirm('Type publish to replace the public document: ')
        except (EOFError, KeyboardInterrupt):
            answer = ''
        if answer != 'publish':
            print('Cancelled; no upload attempted', file=output)
            return 1
        attempted = True
        store.put(document)
        if not (verifier or verify_public)(target, document):
            raise ValueError('Public verification failed')
        print(f"Verified {args.environment} {args.operation} revision={document['revision']}", file=output)
        return 0
    except (Exception, KeyboardInterrupt):
        # Never echo dependency output, credentials, input paths or remote prose.
        if attempted:
            print('Unverified; live state may have changed. No rollback or write retry performed.', file=output)
        else:
            print('Validation or setup failed; no upload attempted.', file=output)
        return 1


if __name__ == '__main__':
    raise SystemExit(run_command())
