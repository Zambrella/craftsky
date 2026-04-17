set dotenv-load := false

default:
    @just --list

# Start the full compose stack in the foreground.
dev:
    docker compose up --build

# Start detached.
dev-d:
    docker compose up -d --build

# Stop and remove containers. Volumes are preserved.
down:
    docker compose down

# Follow logs across all services.
logs:
    docker compose logs -f

# Run the CLI inside the appview container.
migrate *ARGS:
    docker compose exec appview /app/cli migrate {{ARGS}}

ping:
    docker compose exec appview /app/cli ping

tap-status:
    docker compose exec appview /app/cli tap status

# Open a psql session against the dev database.
psql:
    docker compose exec postgres psql -U craftsky craftsky_dev

# Run the Go test suite with the race detector enabled. Tests run on the
# host (the appview image uses a distroless-ish alpine final stage without
# Go) and connect to the compose Postgres via the host-exposed :5432.
# Requires: Go installed locally, and `just dev-d` already running (for
# the real-Postgres integration tests).
test:
    cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5432/craftsky_dev?sslmode=disable go test -race ./...

# Format and vet Go code on the host.
fmt:
    cd appview && gofmt -w . && go vet ./...
