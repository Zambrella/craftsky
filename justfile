set dotenv-load := false

default:
    @just --list

# Create a standard git worktree under .worktrees/.
worktree BRANCH *ARGS:
    ./scripts/worktree-create '{{BRANCH}}' {{ARGS}}

# Prune stale git worktree metadata and empty .worktrees/ directories.
worktree-cleanup *ARGS:
    ./scripts/worktree-cleanup {{ARGS}}

# Start the full compose stack in the foreground.
dev:
    ./scripts/compose-dev up --build

# Start detached.
dev-d:
    ./scripts/compose-dev up -d --build

# Stop and remove containers. Volumes are preserved.
down:
    ./scripts/compose-dev down

# Stop and remove containers, networks, and volumes. This resets dev storage.
reset:
    ./scripts/compose-dev down --volumes --remove-orphans

# Follow logs across all services.
logs:
    ./scripts/compose-dev logs -f

# Run the CLI inside the appview container.
migrate *ARGS:
    ./scripts/compose-dev exec appview /app/cli migrate {{ARGS}}

ping:
    ./scripts/compose-dev exec appview /app/cli ping

tap-status:
    ./scripts/compose-dev exec appview /app/cli tap status

# Compare one DID's Tap/PDS/AppView post state; pass --repair-stale --yes for dev-only cleanup.
tap-repo-check DID *ARGS:
    ./scripts/compose-dev exec appview /app/cli tap repo-check '{{DID}}' {{ARGS}}

# Populate the dev database with deterministic fake posts/comments/replies.
seed-fake *ARGS:
    ./scripts/compose-dev exec appview /app/cli seed fake-posts {{ARGS}}

# Populate the dev database with screenshot-friendly profiles, projects, images, and engagement.
seed-demo *ARGS:
    ./scripts/compose-dev exec appview /app/cli seed demo {{ARGS}}

# Open a psql session against the dev database, or run one-off commands.
#   just psql                 # interactive shell
#   just psql -c '\d'         # pass -c / other args through to psql
psql *ARGS:
    ./scripts/compose-dev exec postgres psql -U craftsky craftsky_dev {{ARGS}}

# Run the Go test suite with the race detector enabled. Tests run on the
# host (the appview image uses a distroless-ish alpine final stage without
# Go) and connect to the current stack's published Postgres port.
# Requires: Go installed locally, and `just dev-d` already running (for
# the real-Postgres integration tests).
test:
    #!/usr/bin/env bash
    set -euo pipefail
    POSTGRES_ADDRESS=$(./scripts/compose-dev port postgres 5432)
    POSTGRES_PORT=${POSTGRES_ADDRESS##*:}
    cd appview
    TEST_DATABASE_URL="postgres://craftsky:dev@localhost:${POSTGRES_PORT}/craftsky_dev?sslmode=disable" go test -race ./...

# Format and vet Go code on the host.
fmt:
    cd appview && gofmt -w . && go vet ./...

# Generate a P-256 private key for OAUTH_CLIENT_SECRET_KEY. Prints to stdout.
# Paste into your local prod-style .env; never commit.
oauth-keygen:
    cd appview && go run ./cmd/cli oauth-keygen

# Regenerate Go types from lexicon/ JSON schemas.
# Two-phase: indigo lexgen (struct shapes) → cbor-gen (CBOR methods).
# Both phases overwrite checked-in files under appview/internal/lexicon/craftsky/.
# Commit the result. See docs/superpowers/specs/2026-04-26-lexicon-codegen-design.md.
lexgen:
    #!/usr/bin/env bash
    set -euo pipefail
    cd appview
    # Pin lexgen to whatever indigo version go.mod resolves to. `go list -m`
    # output is "<module> <version>" — awk strips the module prefix.
    INDIGO_INFO=$(go list -m github.com/bluesky-social/indigo)
    INDIGO_VERSION=$(echo "$INDIGO_INFO" | awk '{print $2}')
    INDIGO_DIR=$(go env GOMODCACHE)/github.com/bluesky-social/indigo@$INDIGO_VERSION
    go run github.com/bluesky-social/indigo/cmd/lexgen@$INDIGO_VERSION \
      --build-file cmd/lexgen/build.json \
      --external-lexicons "$INDIGO_DIR/lexicons/app/bsky/richtext/facet.json" \
      --external-lexicons "$INDIGO_DIR/lexicons/com/atproto/repo/strongRef.json" \
      ../lexicon/social/craftsky
    go run ./cmd/lexgen/cborgen
    gofmt -w internal/lexicon/craftsky

# Drift guard: regenerate and fail if the working tree changes.
# Wire into CI when CI exists.
lexgen-check: lexgen
    git diff --exit-code appview/internal/lexicon/craftsky appview/cmd/lexgen

# Create ignored local Flutter config files from committed examples.
app-env-init:
    #!/usr/bin/env bash
    set -euo pipefail
    for name in local local-android; do
      target="app/config/${name}.env"
      example="app/config/${name}.env.example"
      if [[ ! -f "$target" ]]; then
        cp "$example" "$target"
        echo "Created $target"
      fi
    done

# Run the Flutter app with local config and Flutter's interactive device picker.
app-run *ARGS: app-env-init
    #!/usr/bin/env bash
    set -euo pipefail
    APPVIEW_ADDRESS=$(./scripts/compose-dev port appview 8080)
    APPVIEW_PORT=${APPVIEW_ADDRESS##*:}
    cd app
    flutter run --dart-define-from-file=config/local.env \
      --dart-define="CRAFTSKY_API_BASE_URL=http://localhost:${APPVIEW_PORT}" {{ARGS}}

app-run-chrome: app-env-init
    #!/usr/bin/env bash
    set -euo pipefail
    APPVIEW_ADDRESS=$(./scripts/compose-dev port appview 8080)
    APPVIEW_PORT=${APPVIEW_ADDRESS##*:}
    cd app
    flutter run -d chrome --dart-define-from-file=config/local.env \
      --dart-define="CRAFTSKY_API_BASE_URL=http://localhost:${APPVIEW_PORT}"

app-run-macos: app-env-init
    #!/usr/bin/env bash
    set -euo pipefail
    APPVIEW_ADDRESS=$(./scripts/compose-dev port appview 8080)
    APPVIEW_PORT=${APPVIEW_ADDRESS##*:}
    cd app
    flutter run -d macos --dart-define-from-file=config/local.env \
      --dart-define="CRAFTSKY_API_BASE_URL=http://localhost:${APPVIEW_PORT}"

app-run-ios: app-env-init
    #!/usr/bin/env bash
    set -euo pipefail
    APPVIEW_ADDRESS=$(./scripts/compose-dev port appview 8080)
    APPVIEW_PORT=${APPVIEW_ADDRESS##*:}
    cd app
    flutter run -d ios --dart-define-from-file=config/local.env \
      --dart-define="CRAFTSKY_API_BASE_URL=http://localhost:${APPVIEW_PORT}"

app-run-android: app-env-init
    #!/usr/bin/env bash
    set -euo pipefail
    # atproto's localhost OAuth client only permits loopback redirect URIs.
    # Reverse this checkout's published AppView port into the emulator.
    APPVIEW_ADDRESS=$(./scripts/compose-dev port appview 8080)
    APPVIEW_PORT=${APPVIEW_ADDRESS##*:}
    adb reverse "tcp:${APPVIEW_PORT}" "tcp:${APPVIEW_PORT}"
    cd app
    flutter run --dart-define-from-file=config/local-android.env \
      --dart-define="CRAFTSKY_API_BASE_URL=http://10.0.2.2:${APPVIEW_PORT}"

app-analyze:
    cd app && flutter analyze

app-test *ARGS:
    cd app && flutter test {{ARGS}}

app-build-web ENV="production":
    #!/usr/bin/env bash
    set -euo pipefail
    config="config/{{ENV}}.env"
    test -f "app/$config" || { echo "Missing app/$config. Copy app/$config.example first."; exit 1; }
    cd app
    flutter build web --dart-define-from-file="$config"

app-build-ios ENV="production":
    #!/usr/bin/env bash
    set -euo pipefail
    config="config/{{ENV}}.env"
    test -f "app/$config" || { echo "Missing app/$config. Copy app/$config.example first."; exit 1; }
    cd app
    flutter build ios --no-codesign --dart-define-from-file="$config"

app-build-apk ENV="production":
    #!/usr/bin/env bash
    set -euo pipefail
    config="config/{{ENV}}.env"
    test -f "app/$config" || { echo "Missing app/$config. Copy app/$config.example first."; exit 1; }
    cd app
    flutter build apk --dart-define-from-file="$config"
