#!/usr/bin/env python3
"""Package only public website files, with a verifiable release manifest."""
from __future__ import annotations

import hashlib
import json
import shutil
import subprocess
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ROUTES = {
    "/": "index.html", "/waitlist": "waitlist.html",
    "/privacy": "privacy.html", "/terms": "terms.html",
    "/community-guidelines": "community-guidelines.html",
    "/reporting": "reporting.html", "/copyright": "copyright.html",
}
PUBLIC_FILES = (*ROUTES.values(), "styles.css", "main.js", "waitlist.js", "robots.txt")
ASSET_EXTENSIONS = {".svg", ".png", ".jpg", ".jpeg", ".webp", ".gif", ".ico", ".woff", ".woff2"}


def git(*args: str) -> str:
    return subprocess.check_output(["git", "-C", str(ROOT), *args], text=True).strip()


def build_site(web: Path, commit: str, dirty: bool) -> Path:
    output = web / "dist"
    # Fresh staging prevents stale or development files from entering an upload.
    with tempfile.TemporaryDirectory(prefix=".site-build-", dir=web) as temporary:
        staging = Path(temporary) / "dist"
        staging.mkdir()
        sources = [web / name for name in PUBLIC_FILES]
        sources += sorted(path for path in (web / "assets").rglob("*")
                          if path.suffix.lower() in ASSET_EXTENSIONS)
        for source in sources:
            relative = source.relative_to(web)
            if any(part.startswith(".") for part in relative.parts):
                continue
            if source.is_symlink() or any((web / parent).is_symlink() for parent in relative.parents):
                raise ValueError(f"public file must not be a symlink: {relative}")
            if not source.is_file():
                raise ValueError(f"missing public file: {relative}")
            if source.stat().st_size > 25 * 1024 * 1024:
                raise ValueError(f"public file exceeds Cloudflare's 25 MiB limit: {relative}")
            target = staging / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(source, target)
        manifest = {
            "commit": commit, "dirty": dirty,
            "files": {path.relative_to(staging).as_posix(): hashlib.sha256(path.read_bytes()).hexdigest()
                      for path in sorted(staging.rglob("*")) if path.is_file()},
        }
        (staging / "site-release.json").write_text(json.dumps(manifest, indent=2) + "\n")
        if output.is_symlink():
            raise ValueError("web/dist must not be a symlink")
        if output.exists():
            shutil.rmtree(output)
        staging.rename(output)
    return output


def main() -> int:
    output = build_site(ROOT / "web", git("rev-parse", "HEAD"), bool(git("status", "--porcelain")))
    print(f"Built public website in {output}")
    print("Beta publication does not approve policy drafts; run just online-safety-readiness before full launch.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
