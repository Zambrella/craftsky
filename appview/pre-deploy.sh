#!/bin/sh
set -eu

cd /app
/app/cli --env prod ping
/app/cli --env prod migrate up

# A successful command is not enough: confirm this database reached the newest
# migration shipped in the image before allowing the new server to start.
latest=
for path in migrations/*.up.sql; do
    [ -f "$path" ] || { printf 'no migrations found in /app/migrations\n' >&2; exit 1; }
    name=${path##*/}
    version=${name%%_*}
    case "$version" in
        ''|*[!0-9]*) printf 'invalid migration filename: %s\n' "$name" >&2; exit 1 ;;
    esac
    if [ -z "$latest" ] || [ "$version" -gt "$latest" ]; then
        latest=$version
    fi
done
expected="version=$(expr "$latest" + 0) dirty=false"
actual=$(/app/cli --env prod migrate status)
if [ "$actual" != "$expected" ]; then
    printf 'migration status %s, expected %s\n' "$actual" "$expected" >&2
    exit 1
fi
printf 'migration status: %s\n' "$actual"
