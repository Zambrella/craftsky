# AppView code audit

- Date: 12 August 2026
- Repository snapshot: <code>7615d1774fef9e601e5024693573fdd93b3181d5</code>
- Scope: <code>appview/</code>, its migrations and runtime configuration, plus the smallest client or Compose seams needed to establish impact.

## Executive summary

The AppView has a strong architectural base: its public/private data split is explicit, most public identifiers use the Indigo syntax types, the route policy table makes admission rules inspectable, and the more recent scheduled-post and account-deletion code shows serious attention to idempotency and recovery. The codebase also has substantial tests: 283 Go test files across 554 Go files.

The audit nevertheless found 37 actionable issues:

| Severity | Count |
|---|---:|
| Critical | 1 |
| High | 16 |
| Medium | 16 |
| Low | 4 |

The first changes should be:

1. Close the OAuth SSRF path and inject one hardened HTTP client into every untrusted OAuth/PDS call.
2. Make account and membership departure a single, enforced lifecycle boundary: purge terminal DIDs, revoke sessions, and prevent former members from invoking member-only writes.
3. Replace Tap’s generic retry/drop policy with error classification, a durable quarantine/replay path, and dependency-independent ingestion.
4. Fence all external side effects against account deletion, especially private object upload and automatic-follow writes.
5. Establish a required CI gate with PostgreSQL, MinIO, formatting, static analysis, race testing, and vulnerability scanning.

Because the app is not in production and breaking changes are acceptable, the recommendations favor correcting contracts and schema boundaries directly instead of preserving defective compatibility.

## Method and severity

This was a static review of the full AppView, focused runtime tracing of authentication, routing, Tap ingestion, data lifecycle, scheduled work, notifications, push, and migrations, plus targeted test and analysis runs. Findings were checked against the project architecture documents and authoritative external guidance:

- [Google Go Style Guide](https://google.github.io/styleguide/go/guide): clarity, simplicity, maintainability, comprehensive tests, and mandatory <code>gofmt</code>.
- [Go net/http Server documentation](https://pkg.go.dev/net/http#Server): zero read/write/header timeouts mean no timeout.
- [AT Protocol OAuth specification](https://atproto.com/specs/oauth): externally supplied URLs require SSRF defenses, private-address protection, bounded bodies, and hardened HTTP clients.
- [Indigo Tap documentation](https://pkg.go.dev/github.com/bluesky-social/indigo/cmd/tap): at-least-once delivery, ACK semantics, historical concurrency, and live per-repository barriers.
- [Go vulnerability management](https://go.dev/doc/security/vuln/): source-aware dependency vulnerability analysis with <code>govulncheck</code>.
- [PostgreSQL foreign-key guidance](https://www.postgresql.org/docs/16/ddl-constraints.html): referencing columns are not automatically indexed and can otherwise be scanned during parent deletion.

Severity is based on the potential impact of the defect, not the current launch state:

- **Critical:** unauthenticated compromise of a major trust boundary or plausible system-wide compromise.
- **High:** persistent data loss/privacy failure, credential or authorization failure, or practical service-wide denial of service.
- **Medium:** material correctness, reliability, performance, contract, or defense-in-depth defect with narrower impact.
- **Low:** maintainability, hygiene, or localized efficiency problem.

## Findings at a glance

| ID | Severity | Finding |
|---|---|---|
| AV-001 | Critical | Public OAuth login permits SSRF through discovered endpoints |
| AV-002 | High | Terminal DID deletion leaves a ghost account and private/public data |
| AV-003 | High | Departed members retain usable sessions and member-write access |
| AV-004 | High | Tap ACKs transient indexer failures permanently after six deliveries |
| AV-005 | High | Order-dependent index gates permanently lose posts and interactions |
| AV-006 | High | Account deletion can leave an untracked private-media object |
| AV-007 | High | Automatic-follow can write publicly after departure or deletion |
| AV-008 | High | A long-lived bearer crosses a claimable custom URL scheme |
| AV-009 | High | Concurrent token refresh can revoke a newly valid OAuth session |
| AV-010 | High | Logout leaves the bearer valid when auxiliary cleanup fails |
| AV-011 | High | Authentication-store outages are returned as 401 and erase client sessions |
| AV-012 | High | Missing or unreadable migrations are treated as successful migration |
| AV-013 | High | The build contains 19 reachable known vulnerabilities |
| AV-014 | High | HTTP admission permits slow and large unauthenticated resource exhaustion |
| AV-015 | High | Rate limiting is bypassable and grows memory and auth rows without bound |
| AV-016 | High | Scheduled image validation permits decompression-bomb OOM |
| AV-017 | High | Untrusted OAuth/PDS responses have no overall deadline or size cap |
| AV-018 | Medium | Loopback OAuth handoff is deterministically lost |
| AV-019 | Medium | Partial OAuth callback failure retains unreachable upstream credentials |
| AV-020 | Medium | Configured OAuth expiry does not expire bearer-only access |
| AV-021 | Medium | “All devices” logout leaves other parent OAuth credentials active |
| AV-022 | Medium | OAuth JWKS metadata is derived from the request Host header |
| AV-023 | Medium | Production OAuth configuration fails open to localhost mode |
| AV-024 | Medium | Default development deployment exposes DID impersonation to the LAN |
| AV-025 | Medium | Push lease geometry allows duplicate provider sends |
| AV-026 | Medium | Search query errors are shadowed and can become request panics |
| AV-027 | Medium | Two transactional row loops omit terminal iterator errors |
| AV-028 | Medium | Cascading foreign keys lack usable supporting indexes |
| AV-029 | Medium | Moderation persistence and restoration scheduling are not atomic |
| AV-030 | Medium | Operational duration settings accept zero and negative values |
| AV-031 | Medium | Browser PATCH calls fail CORS preflight |
| AV-032 | Medium | Unknown routes and method mismatches violate the JSON error contract |
| AV-033 | Medium | The default test path skips the database suite, while the DB-enabled suite fails |
| AV-034 | Low | Several indexes exactly duplicate uniqueness indexes |
| AV-035 | Low | Session throttle maps grow for the lifetime of the process |
| AV-036 | Low | Formatting and static-analysis hygiene is not enforced |
| AV-037 | Low | Core API and storage files have grown beyond clear ownership boundaries |

## Critical

### AV-001 — Public OAuth login permits SSRF through discovered endpoints

**Evidence**

- AppView constructs Indigo’s client with its defaults in [internal/app/deps.go](../appview/internal/app/deps.go#L191-L210), then the unauthenticated login handler calls <code>StartAuthFlow</code> in [internal/auth/handlers_session.go](../appview/internal/auth/handlers_session.go#L77-L86).
- The pinned Indigo version is declared in [go.mod](../appview/go.mod#L11). Its metadata resolver uses a public-only transport, but <code>ClientApp.Client</code> is <code>http.DefaultClient</code>. After metadata validation, <code>SendAuthRequest</code> posts directly to <code>pushed_authorization_request_endpoint</code>.
- The pinned metadata validator checks that the PAR endpoint is present, but does not bind it to the authorization-server origin or even require it to be a public HTTPS endpoint. Token and revocation endpoints have the same trust problem.
- The [AT Protocol OAuth security considerations](https://atproto.com/specs/oauth#security-considerations) explicitly call out malicious external URLs, private/internal addresses, SSRF, oversized responses, and slow servers.

**Failure scenario**

An attacker controls a valid public handle/DID, PDS, and authorization-server metadata. The metadata is fetched through the guarded resolver but advertises a PAR endpoint such as a private, loopback, or link-local URL. A call to <code>POST /v1/auth/login</code> then causes AppView to make a blind, unauthenticated-from-the-caller’s-perspective POST into its own network.

**Impact**

This crosses the AppView network trust boundary without a CraftSky account. Depending on deployment topology, it can reach cloud metadata, container administration endpoints, internal HTTP services, or state-changing private APIs. Redirects and DNS changes enlarge the attack surface.

**Recommendation**

Do not rely on endpoint validation inside the current dependency:

1. Inject a single hardened client into Indigo’s <code>ClientApp</code> and resumed sessions.
2. Require every discovered PAR, token, revocation, authorization, resource-server, and PDS URL to use an allowed scheme and public address.
3. Validate DNS results at dial time, block loopback/link-local/private/special ranges, and revalidate every redirect.
4. Bind OAuth endpoints to the issuer/origin rules in the atproto profile.
5. Add phase and total timeouts, response-header limits, and bounded response readers.
6. Add an integration test with public metadata that points PAR at a private test listener and assert that no connection reaches it.
7. Upgrade Indigo if and when a release implements the complete policy, but retain AppView-owned regression tests at this boundary.

## High

### AV-002 — Terminal DID deletion leaves a ghost account and private/public data

**Evidence**

- The terminal identity pipeline registers only notification-actor, Instagram, language, and scheduled-post handlers in [internal/app/deps.go](../appview/internal/app/deps.go#L270-L275).
- Removal of <code>craftsky_profiles</code> and <code>bluesky_profiles</code> exists only in the profile-record delete path in [internal/index/craftsky_profile.go](../appview/internal/index/craftsky_profile.go#L103-L126). A terminal account event does not guarantee individual record deletes.
- Notification terminal cleanup removes rows where the DID is the actor, not every owner/recipient surface, in [internal/notifications/actor_deletion.go](../appview/internal/notifications/actor_deletion.go#L17-L27).
- The explicit deletion flow demonstrates the omitted surface: saved data, pins, customisation, searches, mutes, recipient notifications/preferences, moderation, CraftSky sessions, push subscriptions, and OAuth sessions in [internal/accountdeletion/private_cleanup.go](../appview/internal/accountdeletion/private_cleanup.go#L41-L88).

**Impact**

If a user deletes or deactivates their PDS account outside CraftSky, AppView ACKs the terminal event while retaining current membership, sessions, public projections, relationships, and substantial private data. The result is a ghost member and a privacy/retention failure.

**Recommendation**

Create one idempotent terminal-account purge service that does not require an <code>account_deletion_operation</code>. It should revoke all credentials first, stop/fence external effects, delete owner and recipient private rows, remove every public projection, and be safely replayable. Drive it from the terminal identity event and cover the complete schema in a migration-backed test.

### AV-003 — Departed members retain usable sessions and member-write access

**Evidence**

- Profile deletion removes membership/profile rows but not OAuth or CraftSky sessions in [internal/index/craftsky_profile.go](../appview/internal/index/craftsky_profile.go#L110-L126).
- A <code>CurrentMember</code> middleware exists in [internal/middleware/current_member.go](../appview/internal/middleware/current_member.go#L17-L50), but many member actions are only authenticated in [internal/routes/policy.go](../appview/internal/routes/policy.go#L115-L166): posts, likes, reposts, follows, blocks, reports, ordinary image uploads, saves, notification data, and other private stores.
- <code>POST /v1/posts</code> performs a PDS write in [internal/api/post.go](../appview/internal/api/post.go#L153-L173), while the post indexer discards records from non-members in [internal/index/craftsky_post.go](../appview/internal/index/craftsky_post.go#L72-L79).

**Failure scenario**

A user removes the CraftSky profile record using another atproto client and then reuses the existing CraftSky bearer. AppView can successfully write records to the PDS that its own indexer will later discard, and former members can continue using private/member-only operations.

**Recommendation**

Define the membership authorization matrix explicitly. Apply <code>CurrentMemberRequired</code> to every member-only read/write and external effect, leaving only auth, onboarding, profile creation, and necessary departure/recovery routes available to a non-member. Revoke or quarantine existing sessions when membership disappears. Test that a stale post-departure session never reaches a PDS or private store.

### AV-004 — Tap ACKs transient indexer failures permanently after six deliveries

**Evidence**

- [internal/tap/consumer.go](../appview/internal/tap/consumer.go#L352-L369) ACKs an event after <code>shouldDrop</code>; [the retry tracker](../appview/internal/tap/consumer.go#L466-L481) counts every error without classifying it. With the default five retries, the sixth failure is dropped.
- Defaults are in [internal/app/config.go](../appview/internal/app/config.go#L155-L163) and [docker-compose.yml](../docker-compose.yml#L147-L153); [consumer_test.go](../appview/internal/tap/consumer_test.go#L325-L363) locks in the behavior.
- Tap documents delivery as at least once until ACK, with a default retry timeout. Compose pins Tap 0.1.10 and enables <code>TAP_NO_REPLAY</code> in [docker-compose.yml](../docker-compose.yml#L73-L92).

**Impact**

A database, object-store, PDS, or dependent service outage lasting through six deliveries converts a retryable failure into permanent projection loss. Profiles, posts, relationships, moderation state, and notifications can remain missing with no automatic reconciliation.

The opposite failure also exists: malformed record/identity envelopes in [consumer.go](../appview/internal/tap/consumer.go#L306-L314) and [consumer.go](../appview/internal/tap/consumer.go#L380-L390) bypass both ACK and the retry tracker, so a permanently invalid live event can retry forever and hold Tap’s per-repository barrier.

**Recommendation**

Introduce typed error classes:

- retryable infrastructure errors remain unacknowledged indefinitely, with backoff and alerting;
- permanently invalid payloads are copied to a durable quarantine/dead-letter table, then ACKed;
- every quarantined or terminally failed event has replay/reconciliation tooling;
- outages and malformed-event barriers are covered by integration tests.

### AV-005 — Order-dependent index gates permanently lose posts and interactions

**Evidence**

- [internal/index/craftsky_post.go](../appview/internal/index/craftsky_post.go#L23-L29) documents that a post arriving before membership is permanently dropped; [lines 72–79](../appview/internal/index/craftsky_post.go#L72-L79) return success, so Tap ACKs it. [craftsky_post_test.go](../appview/internal/index/craftsky_post_test.go#L224-L246) asserts this behavior.
- Likes and reposts return success if the actor membership or subject post is not yet present in [internal/index/craftsky_interaction.go](../appview/internal/index/craftsky_interaction.go#L33-L47) and [lines 59–69](../appview/internal/index/craftsky_interaction.go#L59-L69).
- Tap allows historical events from one repository to run concurrently and provides no global ordering across repositories. A like from one repo can therefore precede the subject post projection, and a member’s historical post can precede their profile projection.
- The Bluesky profile repair is a one-shot backfill whose errors are logged and swallowed in [internal/index/craftsky_profile.go](../appview/internal/index/craftsky_profile.go#L92-L102).

**Impact**

On onboarding or resync, existing posts, likes, reposts, counts, and notifications can disappear permanently. The outcome depends on scheduling rather than repository state.

**Recommendation**

Make ingestion independent of projection order. Prefer storing valid source records first and applying membership/visibility on reads. If storage isolation is required, use a durable pending-dependency table and reconcile after profile/post insertion. Make Bluesky profile backfill a durable retryable job. Add historical post-before-profile, interaction-before-subject, and cross-repository interleaving tests.

### AV-006 — Account deletion can leave an untracked private-media object

**Evidence**

- [internal/scheduledposts/media_service.go](../appview/internal/scheduledposts/media_service.go#L72-L99) commits an <code>uploading</code> reservation, releases its transaction advisory lock, performs object-store <code>Put</code>, and then marks the row ready. The reservation and ready compare-and-set are separate at [lines 145–165](../appview/internal/scheduledposts/media_service.go#L145-L165) and [239–258](../appview/internal/scheduledposts/media_service.go#L239-L258).
- Account deletion reads/deletes media rows and queues cleanup without the same object-effect fence in [internal/scheduledposts/account_deletion.go](../appview/internal/scheduledposts/account_deletion.go#L105-L142).
- Cleanup can delete an absent/partial object and finish its job in [internal/scheduledposts/cleanup_processor.go](../appview/internal/scheduledposts/cleanup_processor.go#L158-L190).

**Failure scenario**

Deletion removes the <code>uploading</code> row and cleanup completes before the original object <code>Put</code>. The original upload then creates the object, while its ready-state update affects zero rows. No row or cleanup job remains to discover the private object.

**Recommendation**

Use one shared owner/object effect fence across upload, ready-state CAS, ordinary deletion, and account deletion. A generation-specific staging key plus fenced promotion is another sound design. Add a deterministic barrier test that pauses the first upload between reservation and object creation while deletion completes.

### AV-007 — Automatic-follow can write publicly after departure or deletion

**Evidence**

- [internal/instagram/automatic_follow_worker.go](../appview/internal/instagram/automatic_follow_worker.go#L141-L200) checks membership and policy, then selects a session and invokes the external writer without holding the owner lifecycle/effect lock through the PDS call.
- Membership inactivation takes an owner transaction lock and invalidates pending work in [internal/instagram/account_data.go](../appview/internal/instagram/account_data.go#L50-L129), but it cannot stop a call already in flight.
- The writer executes <code>PutRecord</code> in [internal/followwrite/service.go](../appview/internal/followwrite/service.go#L36-L56). Completion can subsequently lose its local lease, even though the public write succeeded.
- Explicit account deletion removes only <code>social.craftsky.*</code> collections in [internal/accountdeletion/collections.go](../appview/internal/accountdeletion/collections.go#L5-L17), so the late <code>app.bsky.graph.follow</code> is not repaired.

**Recommendation**

Share a session-level owner effect lock between lifecycle/deletion and the final membership/safety recheck plus external PDS call. Deletion must wait for the fence before invalidating sessions and state. Add a mid-write barrier test that removes membership while the provider request is blocked.

### AV-008 — A long-lived bearer crosses a claimable custom URL scheme

**Evidence**

- After OAuth, [internal/auth/handlers_oauth.go](../appview/internal/auth/handlers_oauth.go#L216-L243) embeds the full CraftSky bearer in <code>craftsky:///auth/complete?token=...</code>.
- Android registers an unverified custom scheme in [AndroidManifest.xml](../app/android/app/src/main/AndroidManifest.xml#L28-L33), and iOS does the same in [Info.plist](../app/ios/Runner/Info.plist#L25-L34).
- Android documents that multiple applications can register the same custom scheme, whereas [verified App Links prevent interception](https://developer.android.com/training/app-links/about). Apple similarly documents that [Universal Links cannot be claimed by another app](https://developer.apple.com/library/archive/documentation/General/Conceptual/AppSearch/UniversalLinks.html).

**Impact**

Another installed application can claim the scheme or observe intent/URI telemetry and steal a long-lived bearer, gaining full CraftSky session access. The callback HTML also carries the token without <code>Cache-Control: no-store</code>, a restrictive CSP, or referrer hardening.

**Recommendation**

Return only a short-lived, single-use, device-bound exchange code. Redeem it over TLS and atomically invalidate it. Move to verified Android App Links and iOS Universal Links. Until the bearer is removed from the page, set no-store/no-cache, no-referrer, nosniff, and a nonce/hash-based CSP.

### AV-009 — Concurrent token refresh can revoke a newly valid OAuth session

**Evidence**

- Every request constructs a distinct Indigo session via <code>ResumeSession</code> in [internal/app/deps.go](../appview/internal/app/deps.go#L507-L521).
- The pinned Indigo <code>ClientSession</code> explicitly warns that separate instances for one logical session can clobber data; its refresh mutex is instance-local.
- AppView classifies a losing <code>invalid_grant</code> as terminal in [internal/auth/pds_errors.go](../appview/internal/auth/pds_errors.go#L24-L29), then revokes CraftSky children and deletes the OAuth session in [internal/app/deps.go](../appview/internal/app/deps.go#L670-L688).

**Failure scenario**

Two requests resume the same expired access token. Both use the old refresh token; one succeeds and persists the rotated token, while the other receives <code>invalid_grant</code>. The losing request deletes the newly valid session.

**Recommendation**

Cache/share one session object per <code>(DID, sessionID)</code>, or serialize refreshes with a keyed lock/singleflight. Persist session data with a version and, on <code>invalid_grant</code>, re-read and retry if another refresh changed it. Add a rotating-refresh concurrency test.

### AV-010 — Logout leaves the bearer valid when auxiliary cleanup fails

**Evidence**

- [internal/auth/handlers_session.go](../appview/internal/auth/handlers_session.go#L220-L233) performs push-subscription cleanup before local revocation and returns on cleanup error.
- [handlers_test.go](../appview/internal/auth/handlers_test.go#L354-L374) explicitly asserts that the token remains valid in this case.
- <code>all=true</code> also calls the unbounded upstream OAuth logout before local <code>RevokeAll</code> in [handlers_session.go](../appview/internal/auth/handlers_session.go#L239-L260).

**Impact**

A user can receive a logout failure while the bearer remains usable, including on a shared or lost device. A slow authorization server can also delay the local security action.

**Recommendation**

Revoke the local bearer(s) first in a short transaction. Treat push cleanup and upstream token revocation as idempotent best-effort jobs with retries and observability. The local logout contract should not depend on auxiliary availability.

### AV-011 — Authentication-store outages are returned as 401 and erase client sessions

**Evidence**

- The authentication service distinguishes invalid credentials from database errors in [internal/auth/oauth.go](../appview/internal/auth/oauth.go#L22-L33).
- [internal/middleware/auth.go](../appview/internal/middleware/auth.go#L93-L99) maps every error from <code>Authenticate</code> to 401.
- The Flutter interceptor signs out on every 401 in [sign_out_on_401_interceptor.dart](../app/lib/shared/api/providers/sign_out_on_401_interceptor.dart#L19-L26).

**Impact**

A transient database timeout or outage makes valid credentials appear invalid and causes clients to delete their local session, forcing users through login after recovery.

**Recommendation**

Return 401 only for <code>errors.Is(err, ErrAuthTokenInvalid)</code>. Log and return a retryable 503 (with <code>Retry-After</code> where appropriate) for infrastructure failures. Add an end-to-end test proving that DB failure does not trigger client session invalidation.

### AV-012 — Missing or unreadable migrations are treated as successful migration

**Evidence**

- <code>isMigrationsDirEmpty</code> turns every <code>os.ReadDir</code> error into <code>true</code> in [cmd/cli/migrate.go](../appview/cmd/cli/migrate.go#L128-L141).
- Migration subcommands consequently print “directory is empty” and exit successfully in [cmd/cli/migrate.go](../appview/cmd/cli/migrate.go#L45-L57) and [lines 169–180](../appview/cmd/cli/migrate.go#L169-L180).
- Compose treats successful migration completion as an AppView startup prerequisite in [docker-compose.yml](../docker-compose.yml#L124-L139).

**Impact**

An omitted migration bundle, wrong working directory, or permissions regression starts AppView against a stale or absent schema while deployment reports success.

**Recommendation**

Return <code>(empty bool, err error)</code> and fail closed on missing/unreadable paths. If an empty directory is needed in a test fixture, make that case explicit and never use it as a production shortcut. Test missing, unreadable, empty, and populated directories separately.

### AV-013 — The build contains 19 reachable known vulnerabilities

**Evidence**

<code>govulncheck ./...</code> reported 19 reachable vulnerabilities from the Go standard library and two modules:

- The audit toolchain resolved <code>go 1.26.0</code> from the [go.mod](../appview/go.mod#L3) language/toolchain baseline. The [Dockerfile](../appview/Dockerfile#L2) uses the floating <code>golang:1.26-alpine</code> tag, so the actual release patch level is not reproducible. Reachable findings on the audited toolchain include TLS connection-retention DoS and several <code>html/template</code> escaping/XSS defects; the callback template is in the call graph.
- <code>github.com/jackc/pgx/v5 v5.9.1</code> is below the 5.9.2 fix for [GO-2026-5004](https://pkg.go.dev/vuln/GO-2026-5004).
- <code>google.golang.org/grpc v1.81.1</code> is below the 1.82.1 fix for [GO-2026-6061](https://pkg.go.dev/vuln/GO-2026-6061).

Call-graph reachability is conservative and does not prove every advisory is exploitable in this deployment, but the TLS DoS and template paths are directly relevant.

**Recommendation**

Move the toolchain/container to at least Go 1.26.5 and pin a reproducible image digest, pgx to at least 5.9.2, and gRPC to at least 1.82.1; then rerun unit, integration, race, and vulnerability tests. Make <code>govulncheck</code> a required, regularly refreshed CI gate rather than a launch-only task.

### AV-014 — HTTP admission permits slow and large unauthenticated resource exhaustion

**Evidence**

- The server sets only address and handler in [cmd/appview/main.go](../appview/cmd/appview/main.go#L107-L110). Header, read, write, and idle timeouts are unset; Go documents zero values as no timeout.
- [internal/routes/routes.go](../appview/internal/routes/routes.go#L28-L46) makes body limiting the outermost route wrapper, ahead of authentication and rate limiting.
- [internal/middleware/body_limit.go](../appview/internal/middleware/body_limit.go#L74-L98) reads and retains the complete body. Upload routes allow 15 MiB, and the handler then creates another full payload copy.
- [routes_test.go](../appview/internal/routes/routes_test.go#L501-L518) deliberately locks in body processing before authentication.

**Impact**

Unauthenticated clients can hold connections/goroutines indefinitely with slow headers or bodies and allocate roughly 15 MiB or more per upload-shaped request before rejection.

**Recommendation**

Set <code>ReadHeaderTimeout</code>, <code>IdleTimeout</code>, <code>MaxHeaderBytes</code>, and carefully chosen write/body deadlines. Put cheap connection/IP/content-length admission before body reads, authenticate before expensive buffering, and let each handler own a single bounded stream using <code>http.MaxBytesReader</code> or an equivalent streaming limiter.

### AV-015 — Rate limiting is bypassable and grows memory and auth rows without bound

**Evidence**

- [internal/middleware/rate_limit.go](../appview/internal/middleware/rate_limit.go#L46-L99) stores buckets forever and never evicts them.
- Login’s only limiter key is the caller-supplied device header in [rate_limit.go](../appview/internal/middleware/rate_limit.go#L111-L118). Rotating valid device strings bypasses the ten-per-minute default.
- Protected-route authentication runs before the limiter, so invalid bearer floods hit the database first.
- OAuth auth-request cleanup runs only during callback lookup in [internal/auth/store.go](../appview/internal/auth/store.go#L137-L154); abandoned login starts are never swept by another path. The handoff update also filters an unindexed JSON expression.

**Impact**

An unauthenticated caller can trigger unlimited identity discovery and PAR traffic, grow process memory, create permanent auth-request rows, and drive unbounded database/log load.

**Recommendation**

Add a trusted-proxy-aware IP/global limiter before authentication and retain session/device limits inside it. Use a bounded TTL cache or a shared external limiter, schedule auth-request cleanup independently, store/index <code>request_uri</code> as a normal column, and place hard capacity limits on pending login flows.

### AV-016 — Scheduled image validation permits decompression-bomb OOM

**Evidence**

- The compressed upload may be 15 MiB.
- [internal/api/scheduled_media.go](../appview/internal/api/scheduled_media.go#L86-L106) calls <code>image.Decode</code> directly, with no width, height, total-pixel, frame, or aspect-ratio ceiling.

**Impact**

An authenticated member can submit a compact image declaring extreme dimensions and force a massive pixel allocation, potentially terminating the AppView process. One request can be enough despite route rate limits.

**Recommendation**

Call <code>image.DecodeConfig</code> first, enforce explicit width/height/total-pixel/aspect-ratio limits for every accepted codec, and only then decode. Consider decoding in a separately constrained worker if hostile media remains in-process.

### AV-017 — Untrusted OAuth/PDS responses have no overall deadline or size cap

**Evidence**

- AppView accepts Indigo’s default client in [internal/app/deps.go](../appview/internal/app/deps.go#L191-L210) and passes resumed API clients through in [lines 507–521](../appview/internal/app/deps.go#L507-L521).
- The pinned dependency uses <code>http.DefaultClient</code> for token/PAR/PDS work and decodes or drains response bodies without an AppView-owned cap.
- Inbound request contexts have no AppView deadline because the server has no request timeout.

**Impact**

A slow, malicious, DNS-rebinding, or oversized user-controlled PDS/authorization server can indefinitely retain handlers and workers or exhaust memory. This is distinct from AV-001: even public destinations can be hostile.

**Recommendation**

Use the hardened client described in AV-001 for all federated calls, with dial/TLS/header/idle/total timeouts, bounded bodies, redirect revalidation, per-operation context deadlines, and separate budgets for interactive requests and durable workers.

## Medium

### AV-018 — Loopback OAuth handoff is deterministically lost

**Evidence**

- The callback calls <code>ProcessCallback</code> first and reads handoff metadata afterward in [internal/auth/handlers_oauth.go](../appview/internal/auth/handlers_oauth.go#L124-L170).
- The pinned Indigo callback saves the OAuth session and deletes the auth-request row before returning.
- <code>loadHandoff</code> then queries the deleted row in [internal/auth/handlers_session.go](../appview/internal/auth/handlers_session.go#L156-L169) and falls back to <code>deep_link</code>.

**Impact**

Every successful <code>handoffMode=loopback</code> flow loses its redirect URI; CLI/desktop callers wait while the browser attempts the custom deep link.

**Recommendation**

Load and validate handoff metadata before processing the callback, or move it to a separate single-use table whose lifecycle AppView controls. Add an end-to-end test using the real store and Indigo deletion semantics.

### AV-019 — Partial OAuth callback failure retains unreachable upstream credentials

**Evidence**

<code>ProcessCallback</code> persists the OAuth session before AppView initializes the PDS/profile or creates a CraftSky bearer. Failures at [internal/auth/handlers_oauth.go](../appview/internal/auth/handlers_oauth.go#L194-L221) return without deleting or revoking the newly stored upstream access/refresh tokens and DPoP key.

**Impact**

The database retains live credentials that the user cannot reach or manage through a CraftSky bearer.

**Recommendation**

Treat callback finalization as a saga: defer revocation/deletion of the new OAuth session until local profile initialization and bearer creation commit successfully, or persist an explicit recoverable pending-login state with expiry.

### AV-020 — Configured OAuth expiry does not expire bearer-only access

**Evidence**

- [internal/auth/craftsky_session.go](../appview/internal/auth/craftsky_session.go#L63-L78) authenticates only on token hash and <code>revoked_at</code>.
- Parent absolute/inactivity cleanup runs only inside OAuth <code>GetSession</code> in [internal/auth/store.go](../appview/internal/auth/store.go#L77-L95) and [lines 165–178](../appview/internal/auth/store.go#L165-L178).

**Impact**

A stolen or stale bearer used only for AppView reads can outlive the configured parent OAuth lifetime indefinitely.

**Recommendation**

Make the authentication query authoritative: join the parent session, enforce absolute/inactivity expiry, and revoke/delete expired child sessions atomically. Alternatively, add a reliable sweeper plus an explicit child bearer expiry, but do not depend on a future PDS call.

### AV-021 — “All devices” logout leaves other parent OAuth credentials active

**Evidence**

[internal/auth/handlers_session.go](../appview/internal/auth/handlers_session.go#L239-L260) revokes the presented parent OAuth session and marks all CraftSky children for the DID revoked. It does not enumerate, revoke, or delete other parent OAuth sessions.

**Impact**

The user-visible “all devices” action leaves other access/refresh tokens and DPoP keys active at authorization servers and stored locally, even though their current CraftSky child tokens are disabled.

**Recommendation**

Enumerate every parent session for the DID. Revoke all local child access immediately, then delete local parent credentials and enqueue best-effort authorization-server revocation for each.

### AV-022 — OAuth JWKS metadata is derived from the request Host header

**Evidence**

[internal/auth/handlers_oauth.go](../appview/internal/auth/handlers_oauth.go#L70-L89) builds the confidential client’s <code>jwks_uri</code> from <code>r.Host</code>, although a canonical client origin already exists in the OAuth configuration.

**Impact**

Host-header poisoning or permissive alternate-host routing can make security-sensitive metadata advertise an attacker-influenced key-fetch location, causing broken auth or cache poisoning at authorization servers.

**Recommendation**

Derive every metadata URL from a parsed, validated canonical origin and reject unexpected Host values at the edge/application boundary.

### AV-023 — Production OAuth configuration fails open to localhost mode

**Evidence**

- Production requires a client key only when <code>OAUTH_HOSTNAME</code> is non-empty in [internal/app/config.go](../appview/internal/app/config.go#L346-L349).
- Empty hostname selects the localhost/public client in [internal/auth/config.go](../appview/internal/auth/config.go#L21-L30).
- [internal/app/config_test.go](../appview/internal/app/config_test.go#L366-L378) intentionally accepts a production configuration with no OAuth hostname.

**Impact**

A missing production variable starts successfully with an unusable/weaker localhost callback instead of failing fast. Hostname strings are also interpolated rather than parsed, so values containing a scheme create malformed URLs.

**Recommendation**

For production, require a bare public DNS hostname, confidential key, key ID, callback/client origin, and required scopes. Construct URLs with <code>url.URL</code> and validate the final client metadata at startup.

### AV-024 — Default development deployment exposes DID impersonation to the LAN

**Evidence**

- The server binds <code>0.0.0.0</code> in [cmd/appview/main.go](../appview/cmd/appview/main.go#L107-L110), and Compose publishes the port on all host interfaces in [docker-compose.yml](../docker-compose.yml#L161-L164).
- Dev authentication accepts an arbitrary syntactically valid <code>X-Dev-DID</code> after any invalid bearer in [internal/auth/stacked.go](../appview/internal/auth/stacked.go#L23-L35).

**Impact**

Another device able to reach a developer workstation can impersonate any DID for AppView-private reads and writes. PDS operations that require an OAuth session may fail, but private AppView data remains exposed.

**Recommendation**

Bind published development ports to <code>127.0.0.1</code>. If remote-device development is required, make it an explicit opt-in protected by a high-entropy dev secret or mTLS, not an identity header alone.

### AV-025 — Push lease geometry allows duplicate provider sends

**Evidence**

- Defaults are batch 100, lease 60 seconds, and per-send timeout 10 seconds in [internal/push/dispatcher.go](../appview/internal/push/dispatcher.go#L39-L57).
- All rows receive the same expiry when claimed in [dispatcher.go](../appview/internal/push/dispatcher.go#L123-L189), then are sent serially in [lines 204–278](../appview/internal/push/dispatcher.go#L204-L278).
- A second worker can reclaim expired rows. The old result is fenced from finalizing, but a provider success cannot be undone, and [firebase_sender.go](../appview/internal/push/firebase_sender.go#L32-L47) provides no application idempotency/collapse key.

**Impact**

Worst-case batch time is 1,000 seconds on a 60-second lease. Under slow provider responses and multiple dispatchers, users can receive the same notification more than once.

**Recommendation**

Claim just in time, or use a bounded parallel batch sized from the lease. Carry the actual expiry into each item, cap send contexts to remaining lease minus margin, and renew atomically where needed. Use provider idempotency/collapse semantics when available.

### AV-026 — Search query errors are shadowed and can become request panics

**Evidence**

[internal/api/search_store.go](../appview/internal/api/search_store.go#L731-L815) declares an outer <code>err</code>, then both branches create a new inner <code>err</code> while decoding the cursor. The subsequent <code>pool.Query</code> assigns the inner variable; after the branch, the checked outer error is still nil. Staticcheck reports both assignments as <code>SA4006</code>.

**Impact**

A query error can flow to <code>defer rows.Close()</code> or iteration with an invalid/nil rows value, turning a normal database failure into a panic or aborted response.

**Recommendation**

Use distinct <code>decodeErr</code> variables or check each query immediately in its branch. Add a pool/query failure test for both sort modes and retain <code>SA4006</code> as a required static-analysis check.

### AV-027 — Two transactional row loops omit terminal iterator errors

**Evidence**

- Push claim iterates rows and then leases/commits them without <code>rows.Err()</code> in [internal/push/dispatcher.go](../appview/internal/push/dispatcher.go#L165-L196).
- Notification subscription deactivation does the same before dependent updates and commit in [internal/api/notification_devices.go](../appview/internal/api/notification_devices.go#L121-L149).

**Impact**

A connection, cancellation, or protocol error after a prefix of rows can be mistaken for successful exhaustion and commit a partial operation.

**Recommendation**

Check <code>rows.Err()</code> after every loop and before dependent writes or commit. Add an injected iterator-failure regression test.

### AV-028 — Cascading foreign keys lack usable supporting indexes

**Evidence**

- <code>saved_posts(post_uri)</code> cascades from posts, but existing indexes begin with <code>owner_did</code> in [000024_saved_posts.up.sql](../appview/migrations/000024_saved_posts.up.sql#L1-L30).
- <code>push_deliveries(account_subscription_id)</code> cascades, but its unique index begins with <code>notification_id</code> in [000021_appview_notifications.up.sql](../appview/migrations/000021_appview_notifications.up.sql#L83-L99).
- Likes/reposts have only partial subject indexes, while the FK cascade must also locate soft-deleted rows in [000011_craftsky_interactions.up.sql](../appview/migrations/000011_craftsky_interactions.up.sql#L1-L53).

**Impact**

Deleting a popular post or push subscription can scan entire child tables and hold locks longer as the dataset grows.

**Recommendation**

Add leading indexes on <code>saved_posts(post_uri)</code>, <code>push_deliveries(account_subscription_id)</code>, and non-partial <code>subject_uri</code> indexes for likes/reposts. Validate representative delete plans with <code>EXPLAIN (ANALYZE, BUFFERS)</code>.

### AV-029 — Moderation persistence and restoration scheduling are not atomic

**Evidence**

[internal/api/moderation_store.go](../appview/internal/api/moderation_store.go#L87-L135) inserts a moderation output through the pool and then separately calls the Instagram restoration enqueuer, which performs another independent write in [internal/instagram/restoration.go](../appview/internal/instagram/restoration.go#L83-L100).

**Impact**

If enqueue fails, the API returns an error although the moderation negate is already durable; restoration is absent, and a caller retry may add another output.

**Recommendation**

Insert the output and a durable restoration/outbox record in one transaction. Process that outbox idempotently in the worker.

### AV-030 — Operational duration settings accept zero and negative values

**Evidence**

[internal/app/config.go](../appview/internal/app/config.go#L387-L398) validates duration syntax but not positive semantics. Non-positive Tap ACK timeouts create immediately expired handler contexts; a non-positive reconnect maximum collapses backoff into a hot loop. OAuth expiry/inactivity values have similar nonsensical states.

**Recommendation**

Use a positive-duration parser for deadlines, leases, backoff, and expiry; validate relationships such as inactivity not exceeding absolute lifetime and provider timeout fitting safely inside a lease.

### AV-031 — Browser PATCH calls fail CORS preflight

**Evidence**

[internal/middleware/cors.go](../appview/internal/middleware/cors.go#L21-L39) advertises <code>GET, POST, PUT, DELETE, OPTIONS</code>, while PATCH routes exist for Instagram settings/imports, notification preferences, and saved-post folders in [internal/routes/policy.go](../appview/internal/routes/policy.go#L104-L108) and [lines 132–157](../appview/internal/routes/policy.go#L132-L157).

**Impact**

An allowed browser origin cannot invoke valid PATCH endpoints because the browser rejects preflight.

**Recommendation**

Include PATCH or derive allowed methods from the registered policy table. Add preflight contract tests for every method and credential/origin combination.

### AV-032 — Unknown routes and method mismatches violate the JSON error contract

**Evidence**

[internal/routes/routes.go](../appview/internal/routes/routes.go#L285-L286) falls through to <code>http.NotFoundHandler</code>, and <code>ServeMux</code> generates its own plain-text method errors. The AppView API contract requires the <code>{error, message, requestId}</code> JSON envelope for non-2xx <code>/v1/*</code> responses.

**Impact**

Clients receive inconsistent content types and cannot reliably parse errors for unknown paths or wrong methods.

**Recommendation**

Add a <code>/v1/</code> fallback and method-normalization layer that emits the canonical envelope. Assert body and content type, not status alone.

### AV-033 — The default test path skips the database suite, while the DB-enabled suite fails

**Evidence**

- [internal/testdb/testdb.go](../appview/internal/testdb/testdb.go#L24-L35) silently calls <code>t.Skip</code> when neither database URL is set. Ninety-seven test files use this helper.
- <code>go test ./... -count=1</code> and <code>go test -race ./... -count=1</code> appeared green locally but emitted 484 skip actions.
- The repository contains no CI workflow under <code>.github/workflows</code>.
- A PostgreSQL 15-backed audit run exposed three suite defects: parallel tests race while creating global <code>pg_trgm</code> in [internal/accountdeletion/migrations_test.go](../appview/internal/accountdeletion/migrations_test.go#L17); [worker_acceptance_test.go](../appview/internal/accountdeletion/worker_acceptance_test.go#L21) sends parameterized multi-statement SQL through pgx extended protocol; and [profile_pins_migration_test.go](../appview/internal/db/profile_pins_migration_test.go#L161) makes a timestamp assertion that depends on the host timezone.

**Impact**

The default green result does not validate most persistence behavior, and a future CI service cannot simply enable the database variable and obtain a deterministic pass.

**Recommendation**

Add required CI with PostgreSQL and MinIO. Fail closed when the DB URL is absent in CI, while retaining an explicit unit-only target for fast local use. Make extension setup global/serialized, split parameterized SQL statements, pin test timezone or compare <code>time.Time</code> values, and gate on migrations up/down/up, <code>go test -race</code>, <code>go vet</code>, staticcheck, <code>gofmt</code>, and <code>govulncheck</code>.

## Low

### AV-034 — Several indexes exactly duplicate uniqueness indexes

**Evidence**

- Active likes/reposts each have a unique partial <code>(did, subject_uri)</code> index and an identical non-unique partial index in [000011_craftsky_interactions.up.sql](../appview/migrations/000011_craftsky_interactions.up.sql#L20-L52).
- <code>atproto_follows</code> has a unique constraint on <code>(did, subject_did)</code> plus an identical explicit index in [000012_atproto_follows.up.sql](../appview/migrations/000012_atproto_follows.up.sql#L2-L21).

**Impact**

The duplicates add storage, WAL, cache pressure, and write amplification without adding a new access path.

**Recommendation**

Remove the redundant non-unique indexes after confirming query plans.

### AV-035 — Session throttle maps grow for the lifetime of the process

**Evidence**

[internal/auth/craftsky_session.go](../appview/internal/auth/craftsky_session.go#L26-L41) keeps <code>lastSeenMemory</code> and <code>deviceIDMemory</code> maps. The code itself notes at [lines 106–109](../appview/internal/auth/craftsky_session.go#L106-L109) that the first map only grows during normal operation; neither map evicts revoked or expired sessions.

**Impact**

Long-lived processes accumulate entries for every bearer/session observed. This is smaller and authenticated compared with AV-015, but unnecessary.

**Recommendation**

Use a bounded TTL cache, periodic eviction keyed to session lifetime, or move throttling into an atomic SQL update conditioned on the last timestamp.

### AV-036 — Formatting and static-analysis hygiene is not enforced

**Evidence**

- <code>gofmt -l</code> reports [internal/api/scheduled_post_response.go](../appview/internal/api/scheduled_post_response.go#L9-L25).
- Staticcheck reports the real shadowing defect in AV-026 plus unused functions, unnecessary work, deprecated test API use, and widespread capitalized error strings. Google’s Go guidance requires <code>gofmt</code> and emphasizes clear, maintainable error behavior.
- There is no presubmit workflow to prevent recurrence.

**Recommendation**

Fix the high-signal diagnostics, choose and document the staticcheck set, and enforce <code>gofmt -l</code>, <code>go vet</code>, and staticcheck in CI. Avoid a single bulk style-only rewrite if it would obscure behavioral fixes; this pre-production window is a good time to normalize errors package by package.

### AV-037 — Core API and storage files have grown beyond clear ownership boundaries

**Evidence**

- <code>internal/api/post.go</code>: 2,158 lines
- <code>internal/api/post_store.go</code>: 1,681 lines
- <code>internal/scheduledposts/store.go</code>: 976 lines
- <code>internal/api/search_store.go</code>: 923 lines
- <code>internal/app/deps.go</code>: 750 lines

These files combine multiple endpoint families, query shapes, orchestration rules, and lifecycle concerns. The error-shadowing bug in AV-026 is an example of a small control-flow defect hiding inside a large query method.

**Recommendation**

Split by cohesive capability rather than arbitrary line count: post CRUD/composer, interactions, feed reads, notification reads, search modes, scheduled state transitions, and dependency constructors. Keep transactional invariants together, expose small typed interfaces, and add package-level tests at the new seams.

## Positive observations

The audit also found several practices worth retaining:

- The architecture documents clearly define PDS/AppView ownership and API wire contracts.
- The route policy table centralizes authentication, membership, body, and rate classes, making authorization gaps discoverable and mechanically testable.
- Atproto identifiers are generally parsed at boundaries and carried using Indigo syntax types.
- Most new worker/store paths use contexts, explicit transactions, idempotent state transitions, leases, compare-and-set updates, or advisory locks.
- Account-deletion and scheduled-publication code contains unusually thorough acceptance and recovery tests, even though the cross-boundary races above remain.
- Structured <code>slog</code> logging and JSON error envelopes are used consistently on matched routes.
- The full migration chain applied 1→37, rolled down to zero, and reapplied cleanly against PostgreSQL 15 during the audit.
- <code>go vet ./...</code> passed, and non-database tests passed.
- Only one hand-written Go file failed <code>gofmt</code>, so baseline formatting is otherwise strong.

## Verification record and limitations

| Check | Result |
|---|---|
| <code>go test ./... -count=1</code> | Passed, but with 484 skip actions because no DB URL was present |
| <code>go test -race ./... -count=1</code> | Passed with the same database limitation |
| Focused auth/middleware/routes tests | Passed |
| PostgreSQL 15 migration up/down/up | Passed; final version 37, clean |
| PostgreSQL-backed full suite | Failed on three test-harness/fixture issues described in AV-033 |
| <code>go vet ./...</code> | Passed |
| <code>go run honnef.co/go/tools/cmd/staticcheck@latest ./...</code> | Failed; included AV-026 plus code-quality diagnostics |
| <code>gofmt -l</code> | One file: <code>internal/api/scheduled_post_response.go</code> |
| <code>govulncheck ./...</code> | Failed with 19 reachable vulnerabilities |
| <code>just test</code> | Could not start because the Compose PostgreSQL service was not running |

This is a source audit, not a penetration test or load test. The SSRF, concurrency, and ordering scenarios should be converted into deterministic integration tests before considering them closed. Dependency fixes should be re-scanned on the actual release image because module and standard-library reachability can change after upgrades.

## Suggested remediation sequence

1. **Security boundary:** AV-001, AV-008, AV-009, AV-010, AV-011, AV-013, AV-014, AV-015, AV-016, AV-017.
2. **Lifecycle and authorization:** AV-002, AV-003, AV-006, AV-007, AV-020, AV-021.
3. **Projection correctness:** AV-004 and AV-005, including quarantine, replay, and reconciliation tooling.
4. **Deployment and persistence:** AV-012, AV-027, AV-028, AV-029, AV-030.
5. **API correctness:** AV-018, AV-019, AV-022, AV-023, AV-024, AV-025, AV-026, AV-031, AV-032.
6. **Quality gate and cleanup:** AV-033 through AV-037.

The safest pre-production strategy is to fix the contracts first, then run a destructive local rebuild/reindex from PDS data. There is no benefit in preserving rows, tokens, retry semantics, or API behavior whose current contract is incorrect.
