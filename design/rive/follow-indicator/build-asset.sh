#!/usr/bin/env bash
set -euo pipefail
project_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(git -C "$project_dir" rev-parse --show-toplevel)"
rive "$project_dir" --verify
rive inspect "$project_dir" --summary
# Vector paths and timelines only; there are no scripts to sign.
rive "$project_dir" --once
install -m 644 "$project_dir/build/follow-indicator.riv" "$repo_root/app/assets/animations/follow.riv"
