"""The public dev tunnel must never expose ordinary AppView API/auth routes."""
import http.client
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
import runpy
import threading
import unittest

proxy = runpy.run_path(str(Path(__file__).with_name('revenuecat-webhook-dev')))


class WebhookProxyTests(unittest.TestCase):
    def test_only_signed_webhook_bytes_reach_upstream(self):
        received = []
        class Upstream(BaseHTTPRequestHandler):
            def log_message(self, *_args): pass
            def do_POST(self):
                received.append((self.path, self.rfile.read(int(self.headers['Content-Length'])), dict(self.headers)))
                self.send_response(200); self.end_headers(); self.wfile.write(b'accepted')
        upstream = HTTPServer(('127.0.0.1', 0), Upstream)
        public = HTTPServer(('127.0.0.1', 0), proxy['handler'](upstream.server_port))
        for server in (upstream, public):
            threading.Thread(target=server.serve_forever, daemon=True).start()
        try:
            def request(method, path, body=b'{}', headers=None):
                connection=http.client.HTTPConnection('127.0.0.1',public.server_port)
                connection.request(method,path,body,headers or {})
                result=connection.getresponse(); status=result.status; result.read();connection.close();return status
            self.assertEqual(request('POST','/v1/billing/account'),404)
            self.assertEqual(request('GET','/v1/whoami'),404)
            self.assertEqual(request('POST',proxy['WEBHOOK_PATH']),401)
            self.assertEqual(received,[])
            body=b'{ "signed": "exact whitespace and bytes" }\n'
            headers={'Authorization':'canary-auth','X-RevenueCat-Webhook-Signature':'canary-signature','X-Dev-DID':'must-not-forward'}
            self.assertEqual(request('POST',proxy['WEBHOOK_PATH'],body,headers),200)
            self.assertEqual(len(received),1)
            self.assertEqual(received[0][0],proxy['WEBHOOK_PATH'])
            self.assertEqual(received[0][1],body)
            self.assertEqual(received[0][2]['Authorization'],'canary-auth')
            self.assertNotIn('X-Dev-DID',received[0][2])
        finally:
            for server in (public,upstream):server.shutdown();server.server_close()
