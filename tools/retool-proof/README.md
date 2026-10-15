# Synthetic Retool integration proof

This is a historical prototype. The [current console plan](../../docs/changes/2026-10-14-retool-moderation-console/00-plan.md)
uses a standalone frontend and DID-allowlisted Atmosphere login. Retool draft
verification is no longer required to proceed; retain these fixtures/source as
integration evidence until useful components and tests have been ported.

This disposable server exercises CraftSky's real database-backed moderator
authentication, case reads, command handlers, enforcement output store and
evidence service. It is separate from the production AppView binary and routes.
Only synthetic records and a benign 68-byte PNG are seeded.

The entrypoint accepts only a local database named `craftsky_retool_proof`.
Seeding requires a fresh database. Restarting against an already seeded database
fails rather than resetting data. Proof tokens expire after 24 hours.

## Run locally

From the repository root, with Docker running:

```sh
mkdir -p /private/tmp/craftsky-retool-proof-run
docker run -d --name craftsky-retool-proof-db \
  -p 127.0.0.1:15437:5432 \
  --tmpfs /var/lib/postgresql/data \
  -e POSTGRES_USER=craftsky -e POSTGRES_PASSWORD=dev \
  -e POSTGRES_DB=craftsky_retool_proof postgres:16
docker build -f tools/retool-proof/Dockerfile \
  -t craftsky-retool-proof:local appview
```

Wait for `docker exec craftsky-retool-proof-db pg_isready -U craftsky` to
succeed. For Retool Cloud, start a temporary HTTPS tunnel to port 18147:

```sh
cloudflared tunnel --config /dev/null --no-autoupdate \
  --url http://127.0.0.1:18147 --protocol http2
```

Set `proof_origin` to the HTTPS origin printed by cloudflared, then start:

```sh
docker run -d --name craftsky-retool-proof-api \
  -p 127.0.0.1:18147:8080 \
  -v /private/tmp/craftsky-retool-proof-run:/run/proof \
  -e RETOOL_PROOF_DATABASE_URL='postgres://craftsky:dev@host.docker.internal:15437/craftsky_retool_proof?sslmode=disable' \
  -e RETOOL_PROOF_PUBLIC_ORIGIN="$proof_origin" \
  craftsky-retool-proof:local
```

Without a tunnel, omit `RETOOL_PROOF_PUBLIC_ORIGIN` for local inspection.
The local credential file is `/private/tmp/craftsky-retool-proof-run/credentials.json`
(mode 0600). Configure the `alpha` token directly in Retool resource
authentication; do not put it in app source, commands, logs or this repository.
`beta` lacks evidence-access permission and is used by the backend tests.

Stage 2 also exercises the normal `/v1/admin/identity` route and explicit grants.
The alpha fixture is bound to the approved draft reviewer's Retool user ID and
has moderation read/manage, evidence access and operator-management permissions;
beta has moderation read/manage only. The fixture's identity response is
synthetic. A separate production operator must be provisioned through the
[operator runbook](../../docs/operations/moderation-operators.md).

Use `$proof_origin/openapi.json` as the specification URL. The anonymous spec
contains only schema definitions. Every data/command endpoint requires a token.
The proof-only decision/reversal adapters accept an idempotency key in JSON and
forward it to the unchanged real handler's `Idempotency-Key` header. They operate
only on the seeded case. This works around the observed Retool generated-method
omission of header parameters; it is not a production API change.

## Tests

Use a separate test database so tests never modify the browser fixture:

```sh
docker exec craftsky-retool-proof-db createdb -U craftsky craftsky_retool_proof_test
cd appview
TEST_DATABASE_URL='postgres://craftsky:dev@127.0.0.1:15437/craftsky_retool_proof_test?sslmode=disable' \
TEST_DATABASE_REQUIRED=true \
go test -race ./internal/retoolproof ./cmd/retool-proof -count=1 -timeout=90s
```

Each test creates an isolated migrated schema. The tests cover real apply/replay/
conflict/reversal behavior, actor attribution, cleared strikes, enforcement
outputs, strict adapter decoding, permitted/denied evidence bytes and access
audit records. No external service is needed for these tests.

The independent React regressions run against the retained action hook. Install
test dependencies outside the repository, copy the review source and tests,
then run Node's test loader:

```sh
npm install --prefix /private/tmp/craftsky-retool-ui-tests \
  --no-audit --no-fund react@19.2.7 react-test-renderer@19.2.7 tsx
cp -R tools/retool-proof/retool-source /private/tmp/craftsky-retool-ui-tests/
cp tools/retool-proof/verify-actions.test.tsx /private/tmp/craftsky-retool-ui-tests/
cd /private/tmp/craftsky-retool-ui-tests
node --import tsx --test verify-actions.test.tsx
```

Start that block from the repository root. These tests mock only the API runners
and exercise the real React hook, including uncertain command outcomes and
read-refresh failure. The Go/live resource checks exercise the real backend.

## Stop and reset

Stop the tunnel process, then remove these two dedicated containers:

```sh
docker rm -f craftsky-retool-proof-api craftsky-retool-proof-db
```

The database is tmpfs and disappears with its container. Keep the local
credential file only if intentionally reusing the proof resource credentials
with a fresh fixture; otherwise delete that file and remove or disable the
temporary Retool resources. A new quick tunnel has a new hostname, requiring
resource URL/spec updates. No production resources are involved.

## Retool source

`retool-source/` is a review snapshot of the app-specific MCP-generated source,
not a second writable checkout. Continue changes through the same MCP thread
and refresh this snapshot afterward. Retool's generated runtime, hooks, resource
definitions and dependencies remain platform-owned. Resource credentials are
excluded. See the stage report for verified behavior and remaining gates.
The retained snapshot is stage 1 until the stage 2 generator update can be
inspected and exported following Retool MCP reconnection.
