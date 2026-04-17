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

# Run the Go test suite with the race detector enabled. The container is
# one-shot (--rm) so it does not leave artifacts behind, and it reaches
# postgres via the compose network.
test:
    docker compose run --rm appview go test -race ./...

# Format and vet Go code on the host.
fmt:
    cd appview && gofmt -w . && go vet ./...
