# AV-022, AV-023, AV-024, and AV-030 — Configuration and deployment hardening

- Findings: AV-022, AV-023, AV-024, AV-030
- Severities: Medium (AV-022, AV-023, AV-024, AV-030)
- Priority/order: Configuration boundary; land before any production-style deployment and coordinate with the outbound HTTP and session-lifecycle plans
- Status: Planned
- Sources: [AV-022](../2026-08-12-appview-code-audit.md#av-022--oauth-jwks-metadata-is-derived-from-the-request-host-header), [AV-023](../2026-08-12-appview-code-audit.md#av-023--production-oauth-configuration-fails-open-to-localhost-mode), [AV-024](../2026-08-12-appview-code-audit.md#av-024--default-development-deployment-exposes-did-impersonation-to-the-lan), [AV-030](../2026-08-12-appview-code-audit.md#av-030--operational-duration-settings-accept-zero-and-negative-values)

## Shared implementation strategy

Replace stringly typed, inferred security configuration with one parsed and validated deployment identity. Production starts only with a canonical public HTTPS origin, a complete confidential OAuth client, an explicit allowed-host policy, and positive, internally consistent operational budgets. Development remains convenient by default but is reachable only through loopback-bound Compose ports; any LAN exposure and DID impersonation becomes an explicit secret-protected opt-in.

The configuration loader should produce a complete immutable `Config` whose URL/origin, host policy, environment mode, secrets, and durations already satisfy their semantic invariants. Dependency construction may perform checks that require Indigo key parsing or client-metadata generation, but the process must fail before opening the HTTP listener or starting workers. Handlers then consume precomputed canonical values and never reconstruct security-sensitive URLs from a request.

This is intentionally breaking. Remove `OAUTH_HOSTNAME` and its empty-string localhost fallback in production rather than accepting old malformed values. Change the default Compose host binding. Reject zero/negative or mutually impossible duration settings. With no production users, a loud startup failure and fresh local sign-in are preferable to compatibility aliases that perpetuate unsafe ambiguity.

## Problem to solve

Four configuration defects currently reinforce one another:

- Confidential-client metadata derives `jwks_uri` from attacker-influenced `r.Host` instead of the configured client identity.
- Production with no OAuth hostname is treated as valid and silently selects Indigo's localhost/public client.
- The default Compose port publication exposes a dev auth stack that accepts arbitrary `X-Dev-DID` impersonation to the LAN.
- Several duration variables validate only syntax, so zero or negative deadlines can create immediately canceled work or hot reconnect loops.

The common failure is permissive inference at trust boundaries. Configuration must be explicit and fail closed.

## Desired outcome and invariants

- Every production OAuth URL is derived from one parsed canonical HTTPS origin, never `Host`, forwarded headers, string concatenation, or an unrelated callback variable.
- Production cannot start without the canonical origin, confidential P-256 key, non-empty key ID, required scopes, valid client metadata, and the versioned handoff-receipt encryption key required by AV-008/018/019.
- The canonical production origin is a public DNS origin with no userinfo, query, fragment, path ambiguity, IP literal, localhost/local suffix, or non-HTTPS scheme.
- Requests with an unexpected Host are rejected after request correlation and cheap global/address admission, but before OAuth metadata/callback, route, CORS, authentication, or body work. Proxy behavior is explicit and tested; an alternate Host cannot bypass the outer abuse controls.
- Default Compose-published AppView and development data-service ports bind to `127.0.0.1`, not every host interface.
- `X-Dev-DID` never grants identity in production. In explicitly remote development it additionally requires a high-entropy secret compared in constant time.
- Remote development exposure requires deliberate configuration of both a non-loopback publish address and the protected dev-auth mode; missing either fails closed.
- Every deadline, lease, expiry, polling interval, reconnect cap, and backoff duration that semantically requires progress is strictly positive and within a documented maximum.
- Related timing budgets are validated together, for example inactivity not exceeding absolute lifetime and provider timeout fitting inside its lease with a safety margin.
- No secret is logged, formatted into errors, committed to an environment example, or passed in a URL/command-line argument.

## Scope

### In scope

- Canonical OAuth/public-origin configuration and URL construction.
- Confidential OAuth startup validation and generated metadata validation.
- App-level allowed-Host enforcement and documented reverse-proxy contract.
- Dev port binding defaults and an explicit remote-development opt-in.
- Secret protection for remote `X-Dev-DID` impersonation and CLI support where needed.
- Semantic parsing and cross-field validation for current operational durations plus new durations added by related audit remediations.
- Configuration tests, Compose/environment examples, runbooks, and startup diagnostics.

### Out of scope

- General OAuth/PDS SSRF and response-body limits (AV-001/017), though canonical-origin validation should reuse its URL/address policy where appropriate.
- General inbound HTTP server timeouts (AV-014), though any new timeout config must use the positive-duration parser defined here.
- Rate-limit algorithm/capacity changes (AV-015).
- Mobile verified-link manifests/association files (AV-008/018/019); this plan supplies their canonical origin and validates it.
- Production reverse-proxy/TLS product selection. The plan defines the application contract and required forwarding behavior only.
- A generic secrets manager. Deployment can mount/inject secrets using the chosen platform, but AppView validates their presence and shape.

## Design decisions

### Replace hostname interpolation with a canonical origin

Introduce `OAUTH_PUBLIC_ORIGIN`, for example `https://appview.craftsky.social`, and remove `OAUTH_HOSTNAME`. Parse once with `net/url` into a typed value. An origin is preferable to a bare interpolated hostname because scheme, host, and port policy can be validated explicitly and endpoints can be built with `url.URL`/`ResolveReference` rather than `fmt.Sprintf`.

For production, require:

- scheme exactly `https`;
- a DNS hostname, normalized with lower-case/IDNA policy, with no trailing dot ambiguity;
- no username/password, query, fragment, or path other than empty/root;
- no IP literal, `localhost`, `.localhost`, `.local`, or other non-public/special-use host;
- only the default HTTPS port unless an explicit, justified port policy is added;
- an exact callback of `/oauth/callback`, client ID of `/oauth/client-metadata.json`, JWKS URI of `/oauth/jwks.json`, and verified-link paths derived from the same origin.

Keep development localhost mode explicit through an enum/config branch such as `OAuthModeLocalhost`, not `origin == ""`. Parse `OAUTH_CALLBACK_URL` and require an exact `http://127.0.0.1:<valid-port>/oauth/callback` shape. Production never calls `oauth.NewLocalhostConfig`.

### Precompute and validate metadata

Change `auth.BuildClientConfig` to accept a validated typed deployment config rather than `(hostname string, callback string, ...)`. Parse the P-256 private key and require a non-empty key ID. Require a deduplicated scope list containing `atproto`; reject empty/control-character scopes.

Generate `ClientMetadata` and `JWKSURI` during dependency construction, validate it against `ClientID`, and store the immutable result on `HTTPHandlers` or a dedicated metadata handler. `ClientMetadataHandler` serializes that result and has no reason to inspect `r.Host`.

### Enforce expected Host separately

Add `ExpectedHost` after the common request ID/recovery/safe-logging wrapper and global concurrency plus trusted client-address/global/IP admission, but before routing, CORS, authentication, or body work. In production it accepts only the normalized host from the canonical origin (plus an explicitly configured internal health host only if operations require it). It ignores `Forwarded`/`X-Forwarded-Host` unless a separately trusted proxy contract resolves them before AppView; the simplest production requirement is for the proxy to preserve the canonical `Host`. Host rejection still consumes the same cheap outer budget as all other public traffic and therefore cannot become an admission bypass.

Reject an unexpected or malformed Host with 421 or the documented safe equivalent. For `/v1/*`, retain the standard JSON error envelope; OAuth discovery/callback may use a minimal non-cacheable response. Do not redirect an untrusted Host to the canonical origin.

### Loopback is the default development network boundary

Parameterize the AppView publication independently, for example `${CRAFTSKY_APPVIEW_PUBLISH_HOST:-127.0.0.1}:${CRAFTSKY_APPVIEW_PORT:-18080}:8080`. PostgreSQL and MinIO use separate data-service publish-host variables whose defaults remain `127.0.0.1` and are never inherited from the AppView override. Internal container-to-container addresses remain unchanged. Remote-device testing of AppView must not accidentally publish the database or object store with known development credentials.

Remote-device testing is opt-in by setting the AppView-only publish host plus `APPVIEW_DEV_REMOTE_ACCESS=true`. That mode requires `APPVIEW_DEV_AUTH_SECRET` containing at least 32 random bytes (prefer a 43-character base64url value), and `X-Dev-DID` fallback requires a separate `X-Craftsky-Dev-Authorization` credential. Compare a hash/MAC in constant time and redact it everywhere. Because this credential authorizes DID impersonation, remote mode must also require HTTPS through an approved tunnel/reverse proxy (or mTLS); plain HTTP LAN transport is forbidden. The ordinary OAuth path remains available without this dev impersonation secret.

Local loopback-only mode may retain the header shortcut for CLI ergonomics. If enforcing the publish-address/config relationship inside the process is not reliable under Docker, the Compose wrapper (`scripts/compose-dev`) must derive and pass the matching mode explicitly and refuse contradictory combinations.

### Semantic duration types and relationships

Retain `durationEnv` only for a field where zero/negative has an intentional documented meaning; otherwise use helpers such as `positiveDurationEnv` or `boundedDurationEnv(key, default, min, max)`. Errors name the exact variable and allowed range without printing secrets.

At minimum, validate:

- `TAP_ACK_TIMEOUT > 0` and `TAP_RECONNECT_MAX > 0`, both below documented operational maxima;
- `OAUTH_SESSION_ABSOLUTE_LIFETIME`, `CRAFTSKY_SESSION_INACTIVITY`, `OAUTH_AUTH_REQUEST_EXPIRY`, exchange TTL, confirmation TTL, cleanup grace, `exchange_started` stale-attempt threshold, and `CRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL` are positive;
- confirmation TTL is no longer than exchange TTL; inactivity does not exceed absolute lifetime; authorization-request/exchange TTL is shorter than session lifetime; and activity throttle is shorter than inactivity;
- a finite `exchange_ambiguous` residual horizon is accepted only when tied to a documented provider credential-expiry guarantee; otherwise ambiguous non-secret security evidence is retained for operator/provider follow-up rather than silently expired;
- push polling, lease, and send timeout are positive, with send timeout plus a safety margin shorter than the lease (batch geometry remains AV-025's responsibility);
- worker leases exceed their bounded provider/database operation timeout and backoff initial does not exceed backoff maximum;
- HTTP admission/shutdown/outbound budgets added by AV-001/014/017 are positive and ordered so child operation contexts cannot outlive their owning request/lease.

Use table-driven tests for every `0`, negative, over-maximum, and invalid relationship case. Defaults must pass the same validator as overrides.

## How the shared update closes each finding

### AV-022 — OAuth JWKS metadata is derived from the request Host header

The JWKS URI is derived once from the validated canonical origin and validated with the complete metadata before startup. The request handler never reads `r.Host` to build metadata. Independent expected-Host middleware rejects poisoned/alternate-host requests rather than reflecting them.

### AV-023 — Production OAuth configuration fails open to localhost mode

Production selects an explicit confidential mode and requires all inputs. Missing origin/key/key ID/`atproto` scope, malformed URL, or invalid generated metadata aborts startup. Only explicit dev mode may construct Indigo's localhost client, so an omitted production variable cannot change client type.

### AV-024 — Default development deployment exposes DID impersonation to the LAN

The default host port is loopback-bound. LAN publication requires deliberate configuration and a high-entropy dev credential before `X-Dev-DID` can authenticate. Production strips/rejects all dev-auth configuration. The CLI and documentation no longer imply that identity alone is a credential on a remotely reachable server.

### AV-030 — Operational duration settings accept zero and negative values

All operational durations use semantic positive/bounded parsers and a complete cross-field validator. Impossible settings fail before dependencies, listeners, or workers start, preventing immediately canceled ACK contexts, hot reconnect loops, non-expiring/instantly expiring sessions, and lease/provider budget inversions.

## Unified implementation plan

1. **Write configuration contract tests first.** In `appview/internal/app/config_test.go`, replace the test that accepts production without OAuth settings with required-field cases. Add table-driven origin, scope, host, dev-remote, receipt-key missing/malformed/version, duration-zero/negative/max, and confirmation/exchange relationship tests. In auth handler tests, demonstrate Host poisoning cannot alter metadata.
2. **Introduce typed deployment identity.** In `appview/internal/app/config.go`, replace `OAuthHostname` with a parsed/validated OAuth deployment config containing explicit mode, canonical origin/callback, key material reference/value, key ID, scopes, and expected hosts. Keep URL fields typed or expose defensive copies; do not pass partially validated strings downstream.
3. **Centralize URL validation.** Add focused helpers/tests in `internal/app` or `internal/auth` for canonical public origins and strict loopback callbacks. Reuse public-address/special-host policy from AV-001 where it does not create a package cycle. Normalize before comparison and reject ambiguous encodings rather than repairing them silently.
4. **Refactor OAuth client construction.** In `appview/internal/auth/config.go`, accept the explicit mode/typed URLs, construct endpoints with `url.URL`, require confidential inputs in production, parse the P-256 key, validate scopes, generate client metadata/JWKS URI, and return a complete immutable bundle. Remove `fmt.Sprintf("https://%s/...", hostname)`.
5. **Inject immutable metadata.** Extend `HTTPHandlers` construction in `appview/internal/auth/handlers_oauth.go`, `internal/routes/routes.go`, and `internal/app/deps.go` so `ClientMetadataHandler` serializes the startup-validated document. Remove all `r.Host` URL construction and add no-store/nosniff headers appropriate to metadata/JWKS responses.
6. **Add expected-Host middleware.** Implement and unit-test a small middleware under `appview/internal/middleware/`, and define dev/prod host sets from config. Wire the exact boundary/order from AV-014/015/031/032: selected listener or verified edge active-connection cap and server header ceilings → request ID/recovery/safe logging → global request concurrency → trusted `RemoteAddr`/proxy resolution and global/IP rate limits → `ExpectedHost` → strict canonical escaped-path validation → route catalogue/CORS/authentication/body handling. Cover host with port, case, malformed values, alternate host, direct proxy requests, non-canonical `/v1` paths, and envelope behavior.
7. **Make localhost mode explicit.** Update `LoadConfig`, `BuildClientConfig`, dependency tests, `dev.env`, and `prod.env.example`. Delete `OAUTH_HOSTNAME`. Production missing `OAUTH_PUBLIC_ORIGIN` or secret is an error; development sets its mode explicitly or derives it solely from `EnvDev` plus a valid loopback callback.
8. **Bind Compose ports safely.** Update every host-published development port in `docker-compose.yml` to default to `127.0.0.1`, but use an AppView-specific publish-host variable separate from PostgreSQL and MinIO variables. Update `scripts/compose-dev` port discovery/tests to handle the explicit IP form. Confirm changing only the AppView host leaves data services loopback-only and container health/service traffic on internal names.
9. **Protect remote dev impersonation.** Add `APPVIEW_DEV_REMOTE_ACCESS` and `APPVIEW_DEV_AUTH_SECRET` parsing/validation. Extend the dev auth context/stack in `appview/internal/auth/stacked.go` and `internal/middleware/auth.go` (or a dedicated dev-auth middleware) to require constant-time secret verification in remote mode before accepting `X-Dev-DID`. Require a configured HTTPS tunnel/proxy/mTLS mode for remote access and reject direct HTTP LAN mode. Production clears and rejects the header path.
10. **Update CLI dev requests.** Extend `appview/cmd/cli/request.go` and tests to attach the dev credential only when configured, sourcing it from protected environment/file configuration rather than a command-line flag or URL. Redact request-argument formatting and transport errors. Local loopback behavior remains straightforward.
11. **Add safe remote-development activation.** Update `.env.local.example`, scripts, and README with an explicit opt-in sequence that generates a random secret, changes only the AppView publish host, enables HTTPS tunnel/proxy/mTLS access, and limits firewall scope. The wrapper must reject remote AppView publication without secret and protected-transport mode, and must prove PostgreSQL/MinIO remain loopback-bound. Do not commit a default secret.
12. **Replace permissive duration parsing.** Add semantic duration helpers in `appview/internal/app/config.go`, migrate Tap/OAuth/session/push fields, and reuse the already-positive Instagram parser. Apply the session-integrity plan's breaking names by replacing `OAUTH_SESSION_EXPIRY`, `OAUTH_SESSION_INACTIVITY`, and `CRAFTSKY_SESSION_LAST_SEEN_THROTTLE` with `OAUTH_SESSION_ABSOLUTE_LIFETIME`, `CRAFTSKY_SESSION_INACTIVITY`, and `CRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL`. Add one `Config.Validate()` cross-field pass called for both defaults and overrides. Fold fields introduced by the other audit plans into this validator as they land, including a positive bounded listener active-connection capacity when the in-process AV-014 branch is selected, handoff confirmation TTL ordering, stale `exchange_started` detection, provider-backed ambiguous-residual retention, push lease/send/finalization geometry, scheduled-media object `Put` deadline plus optional backend settlement bound/margin, explicit-deletion PDS settlement behavior, and the terminal Tap pre-ACK budget. Require `terminal fence acquisition budget + fixed tombstone/auth-epoch/component-ledger transaction budget + commit/ACK safety margin < TAP_ACK_TIMEOUT`. Physical terminal purge uses independently bounded batch/lease/backoff settings and is never included in this inequality because owner row cardinality has no hard maximum. A finite ambiguous-attempt or media/deletion tombstone expiry is valid only when the corresponding provider/backend contract is explicitly selected and tested; otherwise retain non-secret attempt/tombstone/job evidence rather than invent an elapsed-time guarantee.
13. **Validate final startup artifacts.** In `newDeps`, validate the generated OAuth metadata against client ID, verify the public origin/expected host agreement, and log only safe mode/origin metadata. Ensure all validation happens before HTTP/worker goroutines start.
14. **Update architecture and operations docs.** Amend OAuth BFF design, environment examples, Compose/README instructions, canonical-link configuration, reverse-proxy Host/TLS contract, duration table, and failure messages. Mark the old env/header behavior as removed rather than deprecated.
15. **Run a residue search.** Before closure, search for `OAUTH_HOSTNAME`, `r.Host` near OAuth metadata, `https://%s/oauth`, unguarded `X-Dev-DID`, non-loopback Compose port publications, and raw `durationEnv` calls on operational fields.

## Data, schema, migration, and reconciliation plan

- No PostgreSQL schema migration is required for AV-022/023/024/030 themselves.
- Changing canonical client ID/origin or confidential-client key can make existing local OAuth sessions inappropriate to retain. Because the app is pre-production, document a one-time local purge of `craftsky_sessions`, `oauth_sessions`, and `oauth_auth_requests` and require fresh sign-in when adopting the new configuration. Let foreign keys/order handle the purge safely; do not automate destructive production deletion in the migration.
- Configuration files and Compose definitions are the migration surface. Remove the old variable rather than support both indefinitely. Startup errors should identify `OAUTH_PUBLIC_ORIGIN` as the replacement.
- If a deployment secret/key is rotated, use the session-integrity cleanup/revocation path where possible, then replace configuration and restart. Do not log or checkpoint the old secret in plan artifacts.
- No public PDS data replay/reindex is required.

## API, client, configuration, and operations impact

- OAuth discovery routes stay at the same paths, but their metadata becomes deterministic for the configured origin. Unexpected Host requests are rejected rather than reflected or redirected.
- Environment breaking changes include replacing `OAUTH_HOSTNAME` with `OAUTH_PUBLIC_ORIGIN`, adding explicit mode/allowed-host/dev-remote settings, adding a secret-mounted versioned handoff-receipt encryption key, and enforcing stricter duration ranges. Update all deployment templates atomically without placing real key material in examples.
- Default Docker host exposure changes from all interfaces to loopback. Developers using a physical phone must opt into AppView-only remote mode through approved HTTPS tunnel/proxy/mTLS transport; PostgreSQL and MinIO remain loopback-only unless independently and deliberately secured.
- Reverse proxies must send the canonical Host, terminate or pass through HTTPS according to deployment design, avoid caching OAuth callback/handoff responses, and reject alternate public hostnames at the edge too.
- Startup logs may state environment, OAuth mode, canonical public hostname, and non-secret budget values. They must never print private key content, dev secret, full database URL, session identifiers, or credential-bearing callback queries.
- Add deployment smoke checks for metadata URLs, JWKS key ID, Host rejection, listener exposure (`127.0.0.1` default), remote-secret enforcement, and effective worker budgets.

## Security, failure, and race considerations

- URL parsing must use `net/url` plus explicit origin checks; parsing success alone does not make a URL safe. Reject backslashes, encoded host confusion, userinfo, Unicode ambiguity, and special/local names according to one normalization policy.
- DNS can change after startup. Canonical-origin validation is not a substitute for AV-001/017 dial-time public-address enforcement on outbound calls.
- Host validation does not trust `X-Forwarded-Host` from arbitrary peers. If a trusted proxy mode is later required, reuse the explicit trusted-CIDR/right-to-left approach already used by `TrustedClientIP` rather than accepting a Boolean `trust proxy` switch.
- Compare the remote-dev secret using a constant-time primitive after hashing/fixed-length normalization. Reject missing/malformed credentials before parsing/using the requested DID where practical.
- Do not use `CRAFTSKY_DEV_DID` as a secret; it is an identity selector. Do not place the new secret in Compose source, mobile assets, shell history, URLs, or CLI flags.
- Never transmit the dev-auth secret over plain HTTP outside loopback. A firewall plus a high-entropy header does not protect the credential from LAN observation.
- A mismatch between publish-host and remote-mode configuration fails the wrapper/startup rather than warning. The safe error includes variable names but not values.
- Duration maxima prevent integer overflow and absurd timers. Cross-field validation uses overflow-safe addition/comparison and must cover boundary equality explicitly.
- Configuration is immutable after startup. Do not dynamically reload half a security bundle while handlers are serving; use validated restart-based changes.
- Handler tests must use a fixed canonical metadata bundle so arbitrary `httptest` Host values cannot change expected output.

## Unified test plan

### Unit tests

- Public-origin matrix: valid HTTPS DNS origin; missing origin; HTTP; IP literal; localhost/local; userinfo; path; query; fragment; malformed port; scheme included in old hostname form; Unicode/trailing-dot/case normalization.
- OAuth completeness matrix: missing/invalid key, empty key ID, missing `atproto`, duplicate/empty scopes, public client in prod, localhost client in prod, and valid confidential client.
- Metadata/JWKS URLs exactly match canonical origin and remain identical under hostile `Host`, `Forwarded`, and `X-Forwarded-Host` headers.
- Expected-Host middleware covers canonical host, canonical default port normalization, alternate host, malformed Host, environment-specific health behavior, and preservation of the request ID/error envelope.
- Dev mode matrix covers loopback default, remote flag without secret, weak secret, secret without remote exposure, valid remote opt-in, wrong/missing header, invalid DID, and production header rejection.
- Every operational duration receives syntax, zero, negative, just-inside, exact-boundary, above-max, and invalid-relationship cases, including rejection of finite media/deletion tombstone expiry without a selected/tested settlement contract and rejection of a fixed terminal tombstone/ledger pre-ACK budget that cannot fit inside `TAP_ACK_TIMEOUT` with margin.

### Integration and deployment tests

- Build production deps from a complete config and assert generated client metadata validates before listener startup.
- Attempt production startup with each required OAuth setting omitted; every case exits non-zero without starting workers/listener.
- Compose config inspection proves published AppView/PostgreSQL/MinIO addresses default to `127.0.0.1`; a socket test from a second LAN namespace/device cannot connect while localhost can. Overriding the AppView publish host must leave PostgreSQL/MinIO loopback-bound.
- Remote opt-in test proves direct HTTP LAN mode is rejected, approved HTTPS transport cannot impersonate with only `X-Dev-DID`, succeeds with the secret, and a wrong secret is indistinguishable from missing authorization.
- Proxy integration sends canonical and poisoned Host values through the intended proxy topology and verifies edge plus application rejection. An alternate-Host flood must consume global/IP limiter and concurrency budgets, return request-ID-bearing `/v1/` envelopes, and never reach route/auth/body work.
- OAuth metadata fetched under alternate Host still either rejects or contains only canonical URLs; it never reflects input.

### Fault and operations tests

- Ensure invalid config exits before DB/object-store/Tap/provider connections where validation does not depend on them.
- Inject the fixed terminal fence/tombstone/auth-epoch/component-ledger commit at the validated deadline and one step over it. In-budget work commits before ACK; an injected over-budget timeout rolls back, sends no ACK, and redelivery retries without partial denial state. Separately seed terminal owners below, at, and far above one physical-purge batch: pre-ACK work and timeout geometry remain catalogue-bounded, terminal predicates hide all retained rows immediately, and durable keyset workers eventually converge. No test or configuration claims to bound total owner inventory.
- Ensure startup errors and structured logs pass a secret scan using known key/dev-secret fixtures.
- Simulate duration boundaries with fake clocks and verify workers back off rather than spin; use a bounded observation rather than a flaky wall-clock sleep.
- Verify a restart with changed canonical origin requires/observes the documented local session reset and fresh login.

### Regression commands

- Run focused Go tests for `internal/app`, `internal/auth`, `internal/middleware`, `cmd/appview`, `cmd/cli`, and Compose/script checks.
- Run the full database-backed race suite and AV-033/036 formatting/static/vulnerability gates after environment/wiring changes.
- Run platform link tests from the AV-008/018/019 plan using the same canonical origin.

## Per-ID traceability and acceptance criteria

### AV-022

- [ ] `ClientMetadataHandler` does not derive any field from `r.Host` or forwarded headers.
- [ ] Client ID, callback, and JWKS URI are generated from one validated canonical origin and validated at startup.
- [ ] Unexpected Host values are rejected after cheap global/address admission but before OAuth/route/auth/body work and cannot poison/cache alternate metadata.
- [ ] Host-poisoning and alternate-Host flood tests pass for OAuth and `/v1/` surfaces, including global/IP admission and non-empty request IDs.

### AV-023

- [ ] Production startup fails when canonical origin, P-256 key, key ID, or required scope is absent/invalid.
- [ ] Production has no code path to `oauth.NewLocalhostConfig`.
- [ ] URL construction uses typed `url.URL` operations; scheme-in-host and other malformed legacy values are rejected.
- [ ] The old test accepting production without OAuth settings is replaced by fail-closed tests.

### AV-024

- [ ] Default Compose AppView port publication is bound to `127.0.0.1` and verified unreachable from another LAN client.
- [ ] LAN/remote development is explicit and cannot use `X-Dev-DID` without a high-entropy dev credential.
- [ ] Remote dev-auth traffic requires approved HTTPS tunnel/proxy or mTLS transport; plain HTTP LAN impersonation mode cannot start.
- [ ] The dev credential is constant-time checked, redacted, uncommitted, and never accepted in production.
- [ ] CLI/docs support the new safe local/remote workflows without putting the secret on a command line or URL.
- [ ] AppView and data services use separate publish-host settings; enabling AppView remote access cannot expose PostgreSQL or MinIO, which remain loopback-bound by default.

### AV-030

- [ ] Tap ACK/reconnect, OAuth/session/auth-request, activity throttle, push, and all newly added operational durations reject zero and negative values.
- [ ] Documented maxima and cross-field relationships are enforced for defaults and overrides, including handoff TTL, stale/ambiguous exchange retention, push lease/send/finalization, object/PDS settlement/tombstone retention, and the fixed terminal tombstone/ledger pre-ACK work fitting inside `TAP_ACK_TIMEOUT` with margin; unbounded physical purge is batch-configured separately.
- [ ] Invalid configuration exits before listener/worker startup with an error naming the exact field(s).
- [ ] Tests prove non-positive reconnect/ACK values cannot create a hot loop or immediately canceled handler context.

## Dependencies and coordination

- **AV-001/017:** share URL/public-address normalization where appropriate and supply outbound dial-time enforcement/timeouts. Canonical-origin checks alone do not close SSRF.
- **AV-008/018/019 grouped OAuth handoff:** consumes `OAUTH_PUBLIC_ORIGIN` for both verified callback paths and adds the versioned receipt-encryption key plus exchange/confirmation/cleanup durations that must use this validator.
- **AV-009/010/011/020/021/035 grouped session integrity:** defines lifetime/lease/provider relationships and fresh-session reset behavior; configuration names/semantics must match.
- **AV-014:** inbound HTTP timeout values must be added to the same positive/bounded configuration validation.
- **AV-015:** trusted-proxy configuration and abuse-limit duration fields should reuse parsing/normalization conventions rather than invent parallel permissive flags.
- **AV-025:** owns push batch/lease geometry; this plan enforces only the shared positive and timeout-within-lease prerequisites.
- **AV-033/036:** CI must exercise production-config failure cases, Compose exposure checks, race tests, and static/format gates.
- **Deployment/platform work:** the canonical HTTPS origin and reverse-proxy behavior must be settled before publishing Android/iOS association files.

## References

- [AppView OAuth BFF design](../superpowers/specs/2026-04-18-appview-oauth-bff-design.md)
- [AppView server scaffold design](../superpowers/specs/2026-04-16-appview-server-scaffold-design.md)
- [AppView API architecture](../superpowers/specs/2026-04-21-appview-api-architecture-design.md)
- [AT Protocol architecture reference](../../atproto-craft-social-app-reference.md#authentication)
- [AT Protocol OAuth specification](https://atproto.com/specs/oauth)
- [Go `net/url`](https://pkg.go.dev/net/url)
- [Go `crypto/subtle`](https://pkg.go.dev/crypto/subtle)
- [Docker Compose port syntax](https://docs.docker.com/reference/compose-file/services/#ports)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
