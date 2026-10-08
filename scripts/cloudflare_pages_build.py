#!/usr/bin/env python3
"""Build the beta website; public launch readiness is a separate owner-run check."""

from __future__ import annotations

import os


def main() -> int:
    branch = os.environ.get("CF_PAGES_BRANCH", "main")
    production_branch = os.environ.get("CF_PAGES_PRODUCTION_BRANCH", "main")
    if branch != production_branch:
        print(
            f"cloudflare-pages-build: preview branch {branch!r}; publication gate skipped"
        )
        return 0

    print(
        "cloudflare-pages-build: beta publication; "
        "run just online-safety-readiness before public launch"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
