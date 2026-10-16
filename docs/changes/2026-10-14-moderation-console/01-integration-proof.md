# Stage 1: Retool moderation integration proof

Archived experiment: the local prototype server, generated UI and related tools
were removed on 16 October 2026. Commands and paths below describe historical
evidence, not supported setup instructions. Use the [operator runbook](../../operations/moderation-operators.md).

Date: 14 October 2026  
Status at 14 October: Private integration draft ready for review; per-user authentication gate remained open

Direction changed on 15 October: the [current plan](00-plan.md) selects a custom
console with DID-allowlisted Atmosphere login. This report preserves the Retool
proof results; its Retool-specific gates are no longer implementation blockers.

## Result and scope

A new private, unpublished [Retool draft](https://craftsky.retool.com/rr/edit/b7ae1ec4-c7e7-11f1-802d-67bd3e9087bc?mcpInitiated=true)
connects to a disposable synthetic backend using an OpenAPI-backed resource.
The existing starter apps were left intact. Draft access was inspected: Doug
Todd and the organisation admin group own it; no wider app access was granted.

The backend uses the real CraftSky moderator authenticator, migrations, case
reads, command handlers, moderation services and enforcement output store. It
contains synthetic operators, an account case and a benign one-pixel PNG. It
has no PDS connection and cannot change production records. No production
routes, tables, configuration or deployment were changed.

At the time of this experiment, the fixture server, integration tests and
OpenAPI contract lived under `appview/internal/retoolproof/`, with the local
runbook and generated app snapshot under `tools/retool-proof/`. These files
were removed on 16 October; use repository history to review the experiment.

## Observed integration results

| Check | Observed result |
|---|---|
| Private draft placement | Created in Drafts without a folder ID; inspected access limited to owner/admin |
| Backend reachability | Temporary Cloudflare HTTPS tunnel reaches a loopback-bound Docker server |
| Authentication and reads | Retool typed identity, queue and detail reads succeeded as `proof-alpha` |
| Reversible command | Applied visibility hiding and a strike, then negated both; real case revision advanced 0 → 1 → 2 |
| Idempotent retry | Same request/key returned `replayed: true`, with no extra revision or effect |
| Stale revision | Backend returned `case_revision_conflict`; fixture remained unchanged |
| Attribution | Retool round trip recorded `proof-alpha`; Go tests also verify reversal attributed to `proof-beta` |
| Enforcement and standing | Go tests verify apply/negate output rows, inactive effects and zero active strikes after reversal |
| Evidence permission | Go tests verify alpha gets 68 PNG bytes and beta gets 403 with no PNG bytes |
| Evidence audit | Tests verify both permitted and denied access records |
| Evidence through Retool | Buffered response is a base64 file object; stream response is a JSON envelope, not direct PNG bytes |
| Per-user custom auth | Not verified; see authentication decision below |

The resource environment is named `production` because the organisation's
default configuration was used after the staging update failed. Its URL and
credentials point exclusively to the synthetic fixture. This label does not
represent a CraftSky production connection.

## Integration adaptations

Retool's generated OpenAPI methods omitted the `Idempotency-Key` header
parameter. A separate REST resource with an attached spec returned
`Your request was blocked.` for both typed and raw calls; its cause remains
unresolved. No production command contract was relaxed.

Two proof-only adapters accept `{expectedRevision, idempotencyKey}` in JSON,
strictly reject unknown fields, scope the action to the one seeded case and
forward the key to the existing real handler. They support only the fixed
synthetic apply/reversal operations. Stage 3 must settle supported header
transport or explicitly design a production command-envelope contract before
implementing arbitrary commands.

The working OpenAPI resource throws errors whose `Error.message` contains the
JSON API envelope. It does not expose HTTP status through that path. App error
normalisation parses `error`, `message` and `requestId`; conflict handling uses
the observed `case_revision_conflict` code. HTTP status is not fabricated.

## Authentication decision and remaining gate

The working proof resource has a fixed synthetic alpha credential in Retool
resource authentication. Backend functions require the trusted Retool
`req.user.id` to match the draft owner's stable ID. The UI distinguishes the
Retool session user from the API operator. This establishes a bounded private
proof, not general per-operator authentication.

A separate custom-auth OpenAPI resource was configured with a form, a
per-user variable and an authenticated identity test URL. The workflow test
reported success, but the typed identity call returned authentication failure
and the resource's authentication-status check reported not authenticated.
The configuration UI explicitly says these variables are usable only with
gRPC, REST API and GraphQL resources. Retool's [custom-auth documentation](https://docs.retool.com/data-sources/guides/authentication/custom)
describes individual credential storage for REST resources, but does not
establish support for the legacy OpenAPI resource used by this proof.

Before shared operator use, prove the REST-plus-OpenAPI resource with per-user
custom auth in the React runtime, including two separate Retool sessions,
correct actor attribution and token revocation. Diagnose the observed blocked
REST execution first. Never derive the AppView actor from a frontend field or
send a claimed actor beside a shared admin credential.

If this Retool resource path cannot support individual credentials, use a
dedicated server-side operator credential/resource per trusted Retool user
with an explicit backend user gate, or a separately reviewed signed-identity
exchange. Neither alternative is implemented or selected for production here.
Stage 1's authentication gate remains open; do not publish or advance to shared
operator rollout on the strength of the fixed alpha proof.

## Evidence-access decision

Retool can show safe metadata and request an authorised evidence operation.
Keep restricted bytes out of ordinary Retool function outputs and builder
fixtures. The tested buffered/stream execution paths wrap binary data, and
Retool function outputs are available during AI-assisted app development.
The benign proof returns only byte counts/type, never the image payload.

Implement evidence viewing as a dedicated AppView operator access flow:
authenticate the individual operator, require permission and a reason, audit
the access, and return bytes with no-store headers. A dedicated viewer must
use its own authenticated session or a narrowly scoped, short-lived access
handoff; never expose a reusable operator token, object-storage key or raw
restricted bytes to Retool's normal query outputs. The final handoff/session
contract and protected-value tests belong to stage 5.

## Validation and runtime lifetime

The following checks passed:

- `go test -race ./internal/retoolproof ./cmd/retool-proof -count=1 -timeout=90s`
  against the separate migrated synthetic test database, including permitted
  and denied evidence access and protected startup diagnostic canaries.
- `go vet ./internal/retoolproof ./cmd/retool-proof`.
- `./scripts/appview-check`: all release gates passed, including the repository
  suites, race/service coverage, Docker/migration/media checks and pinned Tap
  handshake. Artifacts are in `/private/tmp/craftsky-retool-appview-check`.
- OpenAPI validation using `openapi-spec-validator` 0.7.2, against both the
  checked-in document and the document served through the temporary tunnel.
- Final proof Docker image build and startup.
- Live Retool typed reads, apply, unchanged replay, reversal, stale revision
  rejection and benign evidence metadata. Before resetting the proof for UI
  review, the database independently showed revision 2, zero active strikes,
  both command events attributed to alpha and five allowed evidence accesses.
- Retool's final preview compiled and executed identity, queue and detail
  reads with 200 responses. Rendered page text was inspected through the shared
  browser. Full click-through verification and a screenshot capture remain
  unverified: the shared preview's focused interaction/snapshot tools repeatedly
  returned client errors; the standalone sandbox view could render the controls
  but could not complete its parent-mediated function calls.
- Three independent React 19.2.7 hook regressions, using the historical
  `tools/retool-proof/verify-actions.test.tsx` against the exact app source.
  They verify an ambiguous timeout
  preserves the complete request across a newer displayed revision, conflicting
  commands remain blocked through refresh, a refresh failure cannot relabel a
  committed command, and a revision conflict retires its key. The refresh test
  failed before the fix and all three passed afterward.
- `git diff --check` and whitespace inspection of the new files.

The retained app source has a file/hash manifest. It excludes platform runtime
and resource credentials. The final browser fixture was reset to open/revision
0, with no active effects, for operator review. Its current references are
discovered from identity rather than hardcoded in the UI.

The review backend depends on local Docker and the temporary tunnel process.
Tokens/evidence expire 24 hours after seeding. A new tunnel or fixture requires
resource URL/reference updates; functions discover current fixture references
through the identity endpoint. The runbook documents reset and cleanup. No
commit, push, publish or production deployment was performed.
