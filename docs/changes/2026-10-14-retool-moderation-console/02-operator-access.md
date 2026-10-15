# Stage 2: attributable operator access

Status at 14 October: Backend implemented and release checks passed; Retool draft
verification was blocked on MCP reconnection

Direction changed on 15 October: the [current plan](00-plan.md) replaces Retool
with a custom console and DID-allowlisted Atmosphere login. The backend work
below is retained where useful, with human permissions moving to named sets and
browser login moving to OAuth sessions. Retool reconnection is no longer a gate.
The revised authentication design has not been implemented or validated yet.

The approved stage 2 interfaces under test are authenticated AppView admin HTTP
routes and companion CLI credential-management operations. Tests exercise
permissions, individual identity, token lifecycle and audit history through
these interfaces using isolated, migrated PostgreSQL schemas.

This stage removes shared-admin configuration requirements, adds
`GET /v1/admin/identity`, requires explicit permissions on all admin routes and
provides a secure local operator provisioning/rotation/revocation CLI. Existing
restricted-evidence role and helper-assignment boundaries remain authoritative.
Operator changes and credential lifecycle events retain append-only audit
history. Tokens are random, verifier-only in PostgreSQL and delivered in private
files; they never enter app source or normal Retool query results.

## Implemented backend

- Added `GET /v1/admin/identity` with effective permissions, named operator/token
  references, expiry, Retool user binding and helper assignment references.
- Added explicit route permissions for moderation reads/commands, safety reads,
  intake, eligibility, restricted evidence, holds and specialist workflows.
  Authenticated identities without grants are denied. Missing authenticators
  fail closed; the production route configuration has no shared-token fallback.
- Added migration 93 for unique Retool user bindings, the expanded permission
  vocabulary and append-only operator lifecycle/assignment events. Existing
  operators receive no automatic new grants.
- Added `cli operators` bootstrap, create, rotate, token revocation, grant/revoke,
  deactivation, assignment/release and audit operations. Management commands
  require a named administrator with `operator.manage`; actor attribution comes
  from the credential. Only verifier digests are stored in PostgreSQL.
- Private credential delivery is transactional, mode `0600`, refuses overwrite
  and never prints raw credentials. Expiry defaults to 24 hours, capped at
  90 days. Invalid credential-file errors preserve typed diagnostic causes while
  excluding credentials and private paths from serialized output.
- Removed obsolete shared-token environment requirements and changed the local
  moderation helper to consume a CLI-issued private credential file.

The full [operator runbook](../../operations/moderation-operators.md) describes
provisioning, role ceilings, Retool binding, rotation, revocation and recovery.
Operator administration remains in the secure companion CLI in this stage;
there is no browser credential-administration surface or remote provisioning API.

## Retool progress and remaining gate

The private draft continues in the existing stage 1 thread. The selected proof
design uses a dedicated named-operator OpenAPI resource, chosen by a fixed
server-side allowlist using trusted Retool `req.user.id`. Before data access, the
server must match the normal API identity's actor and Retool user binding.
Unmapped or mismatched users must be denied. Credentials remain in resource
authentication, never app source or function outputs.

The new synthetic `CraftSky operator alpha proof` resource's typed
`operator.getOperatorIdentity()` call was observed to return `proof-alpha`,
Retool user ID `2691189`, the individual operator/token references and the
expected effective grants. This proves authenticated resource transport, not
the final app binding guard. The generator was asked to implement that guard,
remove cached identity/case reads and gate actions on effective permissions.

The MCP subsequently returned `MCP authentication required. Reconnect to
continue using this server.` The generated update has not yet been inspected,
tested, exported or approved. The retained source snapshot is still stage 1.
Retool reconnection is required to complete these checks and the stage.

Shared per-user custom authentication is not treated as verified. Separate
Retool sessions for multiple actual operators and per-resource access controls
remain rollout checks. The earlier plain REST resource probe returned a blocked
request despite a successful tool envelope; that path is not selected.

## Verification

Test-first regression slices exercise real authenticated HTTP handlers and
migrated PostgreSQL schemas, including:

- A read-only named operator denied a decision, then explicitly granted access,
  with the resulting event attributed to that operator. A forged actor field is
  rejected. Grant revocation returns 403; token revocation returns 401.
- Rotation invalidates old credentials; expiry and deactivation deny access.
  An ordinary moderator cannot manage operators or exceed its role ceiling.
- Helpers can read only assigned incidents; releasing an assignment denies
  subsequent access. Global queues remain unavailable to helpers.
- Secure output file permissions, refusal to overwrite, rollback cleanup,
  duplicate bootstrap denial, and protected diagnostic canaries.
- The real synthetic case apply/replay/conflict/reversal and evidence tests
  continue to pass with explicit permissions enabled.

Focused race tests, Go vet, shell syntax checks and OpenAPI 3.0 validation passed.
The final `./scripts/appview-check` passed all release gates, including package
and race tests, migration checks, Docker builds/smoke checks and vulnerability
scanning. Artifacts are local at
`/private/tmp/craftsky-stage2-appview-check-final`; the complete log is
`/private/tmp/craftsky-stage2-appview-check-final.log`. The pinned Tap handshake
passed; the gate did not inject events because that Tap version has no safe
deterministic injection API.

No production provisioning, deployment, environment changes, app publishing,
commits or pushes were performed. Stage 3 has not started.
