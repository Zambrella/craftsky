#!/usr/bin/env python3
"""Fail closed before Cloudflare Pages publishes the production branch."""

from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    branch = os.environ.get("CF_PAGES_BRANCH", "main")
    production_branch = os.environ.get("CF_PAGES_PRODUCTION_BRANCH", "main")
    if branch != production_branch:
        print(
            f"cloudflare-pages-build: preview branch {branch!r}; publication gate skipped"
        )
        return 0

    result = subprocess.run(
        [
            sys.executable,
            str(root / "scripts" / "online_safety_readiness.py"),
            "--root",
            str(root),
        ],
        cwd=root,
        check=False,
    )
    if result.returncode != 0:
        print("cloudflare-pages-build: production publication blocked")
    return result.returncode


if __name__ == "__main__":
    raise SystemExit(main())
