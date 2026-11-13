from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
import os
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "scripts"))
from cloudflare_pages_build import ROUTES


ROOT = Path(__file__).resolve().parents[1] / os.environ.get("WEB_TEST_ROOT", ".")


class Handler(SimpleHTTPRequestHandler):
    def translate_path(self, path: str) -> str:
        clean_path = path.split("?", 1)[0].split("#", 1)[0]
        if clean_path in ROUTES:
            return str(ROOT / ROUTES[clean_path])
        return str(ROOT / clean_path.lstrip("/"))

    def log_message(self, format: str, *args: object) -> None:
        del format, args
        pass


ThreadingHTTPServer(("127.0.0.1", 4173), Handler).serve_forever()
