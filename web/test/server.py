from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
import os


ROOT = Path(__file__).resolve().parents[1] / os.environ.get("WEB_TEST_ROOT", ".")
ROUTES = {
    "/": "index.html",
    "/waitlist": "waitlist.html",
    "/privacy": "privacy.html",
    "/terms": "terms.html",
    "/community-guidelines": "community-guidelines.html",
    "/reporting": "reporting.html",
    "/copyright": "copyright.html",
}


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
