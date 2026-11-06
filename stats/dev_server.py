#!/usr/bin/env python3
"""Serve local stats with this checkout's AppView; public files stay unchanged."""
from __future__ import annotations

import argparse
import html
from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlsplit

PUBLIC = Path(__file__).resolve().parent / "public"
PRODUCTION_API = "https://appview.craftsky.social/v1/public/stats"


class StatsDevHandler(SimpleHTTPRequestHandler):
    def __init__(self, *args, api_origin: str, **kwargs):
        self.api_origin = api_origin
        super().__init__(*args, directory=str(PUBLIC), **kwargs)

    def send_head(self):
        # GET and HEAD share the same content and headers. Serve all other
        # files through the standard static handler.
        if urlsplit(self.path).path not in ("/", "/index.html"):
            return super().send_head()
        from io import BytesIO

        source = (PUBLIC / "index.html").read_text()
        body = source.replace(
            f'name="craftsky-stats-api" content="{PRODUCTION_API}"',
            f'name="craftsky-stats-api" content="{html.escape(self.api_origin, quote=True)}/v1/public/stats"',
        ).encode()
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        return BytesIO(body)


def local_origin(value: str) -> str:
    parsed = urlsplit(value)
    if (parsed.scheme != "http" or parsed.hostname not in ("127.0.0.1", "localhost")
            or parsed.username is not None or parsed.password is not None
            or parsed.path or parsed.query or parsed.fragment):
        raise argparse.ArgumentTypeError("expected a local HTTP AppView origin")
    try:
        if parsed.port is None:
            raise ValueError
    except ValueError as error:
        raise argparse.ArgumentTypeError("expected a valid AppView port") from error
    return value


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--api-origin", required=True, type=local_origin)
    parser.add_argument("--port", default=4186, type=int)
    args = parser.parse_args()
    handler = partial(StatsDevHandler, api_origin=args.api_origin)
    with ThreadingHTTPServer(("127.0.0.1", args.port), handler) as server:
        server.serve_forever()


if __name__ == "__main__":
    main()
