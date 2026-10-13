## Architecture Decision Record

- Status: Accepted
- Aspect: OAuth permissions and public Lexicon consent
- Date: 2026-10-13
- Decision: Request granular permissions through a published CraftSky permission set and require fresh consent when an existing grant lacks them

### Why I needed to decide this

Production OAuth discovery still advertises `transition:generic`, despite the
repository defining collection-specific scopes. That gives CraftSky more
repository authority than its features need. Individual collection scopes also
make the provider's consent screen difficult to understand.

### Options I considered

- Keep transitional scopes: simple but grants unnecessarily broad access.
- Enumerate all collections directly: limits authority, but lacks a readable
  description and requires fresh consent whenever a collection is added.
- Publish a namespace-scoped permission set: limits authority to CraftSky records,
  provides a readable consent description, and supports additions through token
  refresh. This is the chosen option.

### What I decided

Publish `social.craftsky.authFull` under the official `craftsky.social` account.
It enumerates the six public record collections: CraftSky actor profile, post,
like, repost, business profile and business event. Create, update and delete
remain available for CraftSky records, including the owner-approved CraftSky
account-deletion flow. Referenced project detail types are not collections and
need no additional repository permission.

Request this set as `include:social.craftsky.authFull`. Providers expand it into
granular `repo` scopes in token responses. Validate the effective collection and
action grants against the embedded permission set, including grouped collections
and split action grants; do not require the literal `include:` scope in tokens. Keep direct, explicit
Bluesky profile create/update, follow and block permissions: these support core
profile/social features and cannot belong in the CraftSky namespace's set.
Keep the supported media MIME allowlist directly in the client scopes; blobs
cannot belong in permission sets. Keep the two video RPC scopes method-specific;
actual service JWTs remain audience-specific and purpose-bound under ADR 012.

Reject transitional scopes and wildcard repository authority in configuration
and initial grants. Before ordinary remote PDS operations, compare stored granular authority
with requested authority under the existing owner/parent fences. Missing required
permissions or old broad scopes invalidate only that parent through the existing
revocation queue. Known typed upstream missing-permission failures require fresh
sign-in, using the existing account-safe client recovery. Generic forbidden
responses must not be mistaken for missing OAuth consent.

Use `/oauth-client-metadata.json` as the canonical confidential client ID. Retain
`/oauth/client-metadata.json` as a discovery alias with its former client ID for
provider caches and revocation; it does not become an alternate login client.
The callback and public JWKS endpoints remain unchanged.

### Publication and rollout

The new NSID requires `_lexicon.craftsky.social` DNS authority and publication in
`com.atproto.lexicon.schema`. Publish and publicly verify before activating the
include scope. Additions to an existing permission set are not guaranteed to
activate after 30 minutes: provider caches may remain stale up to 24 hours, and
new access tokens are needed. Missing-permission recovery remains necessary.

Before changing the client ID, revoke existing production sessions using the
supported CLI and allow the old deployment's revocation worker to drain them.
Credentials issued to the former client ID cannot safely be refreshed or revoked
using the new client ID. Do not delete OAuth rows directly. Include pending
callback and deletion credentials in the drain verification. Close in-flight
login attempts during the maintenance window.

Synchronize the reviewed Render `OAUTH_SCOPES` configuration separately from the
exact-tag application release. Startup rejects stale transitional configuration.
The deployment workflow checks permission publication before deploying and checks
live client ID and scopes afterward, in addition to ordinary public health.

### Consequences

Users with old grants must sign in again. No record shapes change and no PDS
records are deleted by the migration. Tokens and DPoP keys stay server-side.
The permission set adds no new collection authority beyond existing features.

Indigo's Go record generator does not support permission-set generation. Stage
record/type Lexicons for code generation while retaining permission sets in
publication and the embedded validation catalog. Validate correspondence between
the set and actual CraftSky record collections in a source-free test.

### References

- [AT Protocol permissions](https://atproto.com/specs/permission)
- [Lexicon publication](https://atproto.com/specs/lexicon#lexicon-publication-and-resolution)
- [OAuth permissions rollout](../docs/operations/oauth-permissions.md)
- [ADR 012](012-ephemeral-video-service-token-handoff.md)
