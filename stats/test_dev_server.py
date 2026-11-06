"""Regression checks for the actual locally served stats page."""
import threading
import unittest
from functools import partial
from http.server import ThreadingHTTPServer
from urllib.request import Request, urlopen

from dev_server import PUBLIC, PRODUCTION_API, StatsDevHandler


class DevServerTest(unittest.TestCase):
    def setUp(self):
        self.production_source = (PUBLIC / "index.html").read_bytes()
        self.server = ThreadingHTTPServer(
            ("127.0.0.1", 0), partial(StatsDevHandler, api_origin="http://127.0.0.1:18527")
        )
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.origin = f"http://127.0.0.1:{self.server.server_port}"

    def tearDown(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join()

    def test_browser_entrypoints_use_local_api_without_changing_public_source(self):
        for path in ("/", "/index.html", "/?local=1"):
            with self.subTest(path=path), urlopen(self.origin + path) as response:
                body = response.read().decode()
                self.assertEqual(response.status, 200)
                self.assertEqual(response.headers["Cache-Control"], "no-store")
                self.assertIn('content="http://127.0.0.1:18527/v1/public/stats"', body)
                self.assertNotIn(PRODUCTION_API, body)
        self.assertEqual((PUBLIC / "index.html").read_bytes(), self.production_source)
        self.assertIn(PRODUCTION_API.encode(), self.production_source)

    def test_head_and_static_assets_work(self):
        with urlopen(self.origin + "/") as response:
            size = len(response.read())
        with urlopen(Request(self.origin + "/", method="HEAD")) as response:
            self.assertEqual(int(response.headers["Content-Length"]), size)
            self.assertEqual(response.read(), b"")
        with urlopen(self.origin + "/main.js") as response:
            self.assertEqual(response.read(), (PUBLIC / "main.js").read_bytes())


if __name__ == "__main__":
    unittest.main()
