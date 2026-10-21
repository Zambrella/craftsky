"""Disposable compiled Flutter A/B server and independent anonymous status origin."""
import json
import mimetypes
import os
from pathlib import Path
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import threading
from urllib.parse import unquote, urlsplit

ROOTS = {key: Path(os.environ['CRAFTSKY_STATUS_BUILD_' + key]).resolve() for key in ('A', 'B')}
STATE = {'build': 'A', 'minimum': None, 'unavailable': False, 'messageRevision': None}
REQUESTS = []
LOCK = threading.Lock()
APP_PORT = int(os.environ.get('CRAFTSKY_STATUS_APP_PORT', '4175'))
STATUS_PORT = int(os.environ.get('CRAFTSKY_STATUS_HTTP_PORT', '4176'))

class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass

    def respond(self, body, mime='application/json', status=200):
        self.send_response(status)
        self.send_header('Content-Type', mime)
        self.send_header('Content-Length', str(len(body)))
        self.send_header('Cache-Control', 'no-store')
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('X-Content-Type-Options', 'nosniff')
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        if self.server.server_port != APP_PORT or self.path != '/__control':
            return self.respond(b'{}', status=404)
        value = json.loads(self.rfile.read(min(int(self.headers.get('Content-Length', '0')), 1024)))
        with LOCK:
            STATE.update({key: value[key] for key in STATE if key in value})
            REQUESTS.clear()
        self.respond(b'{}')

    def do_GET(self):
        path = urlsplit(self.path).path
        with LOCK:
            state = dict(STATE)
        if self.server.server_port == STATUS_PORT:
            if path != '/app.json':
                return self.respond(b'{}', status=404)
            with LOCK:
                REQUESTS.append({key: self.headers.get(key) is not None for key in ('Authorization', 'Cookie', 'Referer', 'X-Device-ID', 'X-Account-DID')})
            if state['unavailable']:
                return self.respond(b'{}', status=503)
            document = {'schemaVersion': 2, 'revision': 'local-harness'}
            if state['minimum']:
                document['minimumVersions'] = {'web': state['minimum']}
            if state['messageRevision']:
                document['message'] = {'revision': state['messageRevision'], 'minVersion': '0.0.0',
                    'title': 'Storage test notice', 'description': 'Public browser storage fixture'}
            return self.respond(json.dumps(document).encode())
        if path == '/__requests':
            with LOCK:
                return self.respond(json.dumps(REQUESTS).encode())
        root = ROOTS[state['build']]
        candidate = (root / unquote(path).lstrip('/')).resolve()
        if not candidate.is_relative_to(root):
            return self.respond(b'{}', status=404)
        if not candidate.is_file():
            if '.' in Path(path).name:
                return self.respond(b'{}', status=404)
            candidate = root / 'index.html'
        mime = {'.js': 'text/javascript', '.wasm': 'application/wasm', '.json': 'application/json'}.get(candidate.suffix)
        self.respond(candidate.read_bytes(), mime or mimetypes.guess_type(str(candidate))[0] or 'application/octet-stream')

if __name__ == '__main__':
    status = ThreadingHTTPServer(('127.0.0.1', STATUS_PORT), Handler)
    threading.Thread(target=status.serve_forever, daemon=True).start()
    app = ThreadingHTTPServer(('127.0.0.1', APP_PORT), Handler)
    try:
        app.serve_forever()
    finally:
        status.shutdown()
