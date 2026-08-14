# AV-001 / AV-017 — Federated HTTP trust boundary

- **Findings:** AV-001, “Public OAuth login permits SSRF through discovered endpoints”; AV-017, “Untrusted OAuth/PDS responses have no overall deadline or size cap”
- **Severity:** Critical / High
- **Priority/order:** 1 — land before enabling any non-local OAuth/PDS traffic; coordinate with the verification gate in AV-033 and AV-036
- **Status:** Planned
- **Source:** [AV-001](../2026-08-12-appview-code-audit.md#av-001--public-oauth-login-permits-ssrf-through-discovered-endpoints) and [AV-017](../2026-08-12-appview-code-audit.md#av-017--untrusted-oauthpds-responses-have-no-overall-deadline-or-size-cap)

## Shared implementation strategy

Build one AppView-owned federated HTTP boundary and make every OAuth, authorization-server, resource-server, and PDS request pass through it. The boundary must combine URL-policy validation, dial-time public-address enforcement, redirect revalidation, bounded response bodies, connection limits, and operation deadlines. Merely swapping `http.DefaultClient` for a client with `Timeout` is insufficient: it would leave discovered endpoint validation, DNS rebinding, redirects, and response allocation as separate gaps.

Keep Indigo as the atproto OAuth implementation. AppView should wrap and configure it through supported `oauth.ClientApp` and `atclient.APIClient` seams; it must not introduce a generic OAuth library or fork protocol logic into handlers. AppView-owned policy and regression tests remain authoritative even after an Indigo upgrade.

The boundary should expose a small set of purpose-specific clients backed by the same transport and destination policy:

- **OAuth discovery/metadata:** HTTPS-only, public destinations, strict metadata size limit, no cross-origin redirect.
- **OAuth PAR/token/revocation:** endpoint validated against the discovered issuer and AT Protocol origin rules before use; small response limit.
- **Authenticated and anonymous PDS JSON:** destination derived from a validated DID/OAuth session, public-only dialing, bounded JSON responses.
- **PDS upload response:** the request body retains the existing inbound media limit, while the response is independently bounded.
- **Interactive and worker execution profiles:** the same security policy with different explicit total deadlines; durable workers retry a timed-out attempt rather than holding a lease indefinitely.

Because the app is pre-production, replace the current permissive contracts directly. Do not retain a compatibility path using `http.DefaultClient`, allow private destinations for old rows, or silently fall back when policy configuration is absent.

## Finding closure

### AV-001 — Public OAuth login permits SSRF through discovered endpoints

The shared update closes AV-001 by validating every metadata-supplied URL before it can become an HTTP request and by enforcing the same decision again at dial and redirect time. Specifically:

- Parse URLs canonically and reject non-HTTPS schemes, userinfo, fragments, malformed hosts, unexpected ports, and endpoint/origin relationships disallowed by the AT Protocol OAuth profile.
- Resolve at dial time and reject loopback, private, link-local, unspecified, multicast, documentation, carrier-grade NAT, IPv4-mapped private IPv6, and other non-public/special-use addresses. Dial an accepted resolved address while retaining the original hostname for TLS verification so a second resolution cannot redirect the connection.
- Re-run destination policy for every redirect, cap redirect count, and reject HTTPS-to-HTTP or cross-origin redirects where the protocol does not explicitly allow them.
- Validate the authorization URL returned to the browser against the discovered authorization-server metadata before returning it from `POST /v1/auth/login`.
- Apply this policy to PAR, token, revocation, protected-resource metadata, authorization-server metadata, resumed session endpoints, anonymous PDS lookups, and OAuth-session `HostURL`; do not treat a previously persisted URL as trusted forever.

The public login endpoint must return the existing camelCase error envelope without revealing the rejected address, DNS result, or internal topology. Logs and metrics should use bounded categories such as `destination_rejected`, `redirect_rejected`, and `issuer_mismatch`, never tokens, raw query strings, or DPoP material.

### AV-017 — Untrusted OAuth/PDS responses have no overall deadline or size cap

The same boundary closes AV-017 by making both time and memory finite for public-but-hostile peers:

- Configure connect, TLS handshake, response-header, idle-connection, and expect-continue timeouts on the shared transport, plus a non-zero total deadline for each operation.
- Bound response bodies before Indigo or AppView decodes them. Use deliberately small ceilings for metadata and token/PAR responses, a separate bounded ceiling for ordinary PDS JSON, and a narrow upload response ceiling. A limit breach must return a typed `response_too_large` error without draining an unbounded body.
- Cap redirects, per-host connections, idle connections, and idle lifetimes so a single PDS cannot consume the whole pool.
- Add context deadlines at the call sites that own semantics: login/callback requests use an interactive budget; scheduled publishing, account deletion, and backfill use a per-attempt worker budget that fits inside their lease/retry model. Apply the same worker budget to Instagram automatic follow only if AV-007's approved branch retains that PDS effect; the strict branch instead proves no background PDS call exists.
- Close bodies on every path and preserve cancellation through Indigo adapters. Timeouts, cancellation, malformed JSON, and oversize responses must be classified distinctly from terminal OAuth credential rejection.

Security ceilings should be hard maximums in code. Deployment configuration may lower timeouts or limits but must validate positive values and must not disable the boundary with zero or negative values.

## Desired outcome and invariants

- No caller-controlled OAuth/PDS URL can cause AppView to connect to a non-public or protocol-invalid destination.
- DNS changes and redirects cannot bypass the decision made for the original URL.
- Every federated request has a finite connection/header/body/total budget and every decoded response has a finite byte ceiling.
- OAuth remains DPoP-capable and AppView remains the only holder of PDS access/refresh tokens.
- Public writes still go to the owner’s PDS; the boundary changes transport safety, not public/private ownership.
- Durable jobs remain replayable: a timeout or dependency outage records a retryable attempt, while a policy violation is terminal/quarantined as appropriate for that job.
- No error or diagnostic includes bearer tokens, OAuth tokens, DPoP keys, response bodies, private IPs, or unbounded remote URLs.

## Scope

### In scope

- A new federated destination/transport package, likely `appview/internal/federatedhttp/`.
- Wiring hardened clients into `appview/internal/app/deps.go`, Indigo `oauth.ClientApp.Client`, its `Resolver.Client`, resumed `ClientSession`/API clients, and `auth.AnonymousPDSClient`.
- Endpoint and persisted-session validation in `appview/internal/auth/`.
- Bounded decoding in `appview/internal/auth/pds_client_indigo.go` where Indigo does not expose an adequate response-limit hook.
- Per-operation deadlines at interactive handlers and external-effect workers.
- Configuration, observability, failure translation, and regression tests for the boundary.

### Out of scope

- Replacing Indigo with a generic OAuth library.
- Changing the BFF rule or exposing PDS credentials to Flutter.
- A generic outbound proxy or unrestricted allow-list bypass.
- Inbound HTTP admission/rate limiting, covered by AV-014 and AV-015.
- Dependency-version remediation itself, covered by AV-013.

## Design decisions

1. **One policy engine, several bounded clients.** Reuse the same destination validator and transport; vary only response and operation budgets by traffic purpose.
2. **Validate twice.** Validate discovered/persisted URLs before request construction and validate the actual network destination at dial/redirect time. Either check alone is incomplete.
3. **Fail closed.** Missing policy, resolution ambiguity, mixed public/private DNS answers, unsupported schemes, or invalid endpoint relationships stop the request.
4. **Pin a validated address per connection.** Do not validate one DNS answer and let the default dialer resolve the hostname again.
5. **Prefer explicit caps over generic decode.** Introduce a limit-aware response wrapper or adapter so `json.Decoder` cannot read past the purpose-specific ceiling.
6. **Separate security ceilings from tunables.** Code owns non-disableable upper bounds; validated configuration may choose stricter operational values.
7. **Use typed failures.** Destination-policy violations, timeout/cancellation, response-too-large, upstream HTTP failures, malformed responses, and expired credentials must remain distinguishable.

## Unified implementation plan

1. Write failure-first tests for malicious discovered PAR/token/revocation/PDS endpoints, DNS rebinding, redirects, slow responses, and oversized bodies.
2. Add `internal/federatedhttp` types for request purpose, URL validation, endpoint/issuer validation, address classification, dial-time resolution, redirect policy, bounded response bodies, and typed errors.
3. Clone and harden `http.DefaultTransport` rather than constructing a zero-value transport. Set explicit dial/TLS/header/idle limits, connection-pool bounds, and proxy behavior. Define how trusted deployment proxies are handled; never let environment proxy settings route protected destinations unexpectedly.
4. Add purpose-specific client constructors with explicit total deadlines and response ceilings. Suggested starting ceilings are 64 KiB for discovery metadata, 256 KiB for PAR/token/revocation and upload responses, and a bounded multi-megabyte limit for paged PDS JSON. Confirm each value against current request/page shapes before committing it.
5. In `internal/app/deps.go`, construct the boundary once. Assign the OAuth client to `oauthApp.Client`, assign the discovery client to `oauthApp.Resolver.Client`, and inject the PDS profile into anonymous and resumed PDS clients.
6. In `internal/auth`, validate all fields received from authorization-server/protected-resource metadata and all session URLs loaded from `oauth_sessions.data`. Apply the exact issuer/origin relationships from the AT Protocol OAuth specification.
7. Adapt `AnonymousPDSClient` to accept an injected client/factory rather than allocating `&http.Client{Timeout: ...}` for each call. Ensure DID-document resolution and the resulting PDS endpoint both use the boundary.
8. Update `IndigoPDSClient`, especially `UploadBlob` and record-listing paths, so response decoding observes explicit byte limits even if the upstream dependency does not.
9. Add operation-deadline helpers at `LoginHandler`, `CallbackHandler`, profile initialization/backfill, scheduled publication, and account deletion. If AV-007 retains automatic following, apply the helper to its call and feed timeout ambiguity into the approved durable attempt/reconciliation path; otherwise add a capability test proving that call site is absent.
10. Map boundary errors to the standard `/v1/*` envelope and bounded OAuth callback HTML. Do not expose target details. Add stable structured metrics for request purpose, result category, and latency.
11. Add validated configuration for operational budgets in `internal/app/config.go` and environment documentation. Reject zero/negative values and impossible worker deadline/lease relationships, coordinating with AV-030.
12. Run focused auth/PDS/worker tests, the PostgreSQL-backed suite, race tests, static analysis, and `govulncheck`. Verify with an instrumented private listener that rejected SSRF attempts make zero connections.

## Data, schema, migration, and reconciliation plan

No schema migration is required for the core transport boundary. Existing OAuth session JSON may contain endpoints that no longer pass validation; because breaking changes are acceptable, reject those sessions, revoke their CraftSky children, and require a fresh login rather than grandfathering unsafe endpoints.

Before rollout, provide a dry-run diagnostic command or startup report that counts persisted sessions by validation result without logging endpoints or credentials. The rollout may delete invalid local OAuth/CraftSky sessions after review. There is no public-record migration and no PDS data rewrite.

## API, client, configuration, and operations impact

- `POST /v1/auth/login` keeps its request/response shape. Invalid or unreachable federated destinations continue to use the documented JSON error envelope, with no internal target details.
- Existing sessions pointing at newly rejected destinations become unauthorized only after their local credentials are safely revoked/deleted; Flutter should follow normal invalid-session handling.
- Add explicit, positive timeout/limit settings to environment examples only where operators need to tune below code ceilings. Defaults must be secure.
- Emit counters for rejected destinations/redirects, timeouts, response-limit breaches, and upstream failures by request purpose. Alert on sustained failures without labeling metrics by raw hostname, DID, token, or URL.
- Document that private/local PDS endpoints are intentionally unsupported by production AppView. If local federation testing is necessary, use an explicit dev-only injected test transport bound to loopback, not a production bypass flag.

## Security, failure, and race considerations

- Classify IPv4-in-IPv6 and all DNS answers, not only the first address. Mixed-answer hosts fail closed.
- Preserve TLS verification against the original hostname when dialing a validated IP; never set `InsecureSkipVerify`.
- Revalidate each redirect and strip sensitive headers on cross-origin redirects. Prefer rejecting protocol-unexpected cross-origin redirects entirely.
- Do not retry non-idempotent PAR/token operations automatically at the transport layer. Let the owning OAuth flow decide whether a new attempt is safe.
- Bound error bodies as well as success bodies; hostile peers often place large data on non-2xx paths.
- A context timeout must not be translated to `ErrPDSSessionExpired`; otherwise a network outage could revoke valid credentials.
- Shared transports/clients must be safe for concurrent use and closed only during process shutdown, not per request.

## Unified test plan

### Unit tests

- URL canonicalization and endpoint/issuer relationship tables.
- Public and rejected IPv4/IPv6 ranges, IPv4-mapped IPv6, mixed DNS answers, userinfo, fragments, schemes, and ports.
- Redirect count, downgrade, cross-origin, and private-target rejection.
- Purpose-specific response limits on success and error bodies, including exactly-at-limit and one-byte-over cases.
- Timeout/error classification and redaction.

### Integration tests

- A public-looking authorization-server metadata fixture advertises private PAR, token, and revocation listeners; assert each is rejected and the listeners receive zero connections.
- A resolver changes from a public answer to loopback between validation and connection; assert the validated dial path cannot rebind.
- A permitted public fixture redirects to private/link-local and is rejected before connection.
- Slow header, slow body, never-ending body, oversized JSON, malformed JSON, and connection-stall fixtures terminate within their budgets without leaked goroutines.
- Run real `StartAuthFlow`, callback/token exchange, resumed session, anonymous backfill, record read/write/list, and blob-upload response paths through the injected boundary.

### Worker and concurrency tests

- A timed-out scheduled publication and account deletion release/record their leases according to durable retry contracts. AV-007 testing is branch-specific: retained automatic follow records timeout ambiguity without an unauthorized repeat, while strict removal proves no background PDS request can time out.
- Concurrent requests reuse the shared client safely under `go test -race`.
- Cancellation closes the response and connection promptly and does not revoke a valid OAuth session.

### End-to-end/operations tests

- Complete login and a PDS write against an approved test PDS over TLS.
- Verify metrics contain bounded categories and no endpoint query strings, credentials, DIDs, response bodies, or private addresses.
- Run the invalid-session dry run, remove unsafe local sessions, and confirm a clean re-login succeeds.

## Per-ID acceptance criteria

### AV-001

- [ ] Every discovered or persisted OAuth/PDS URL is validated against scheme, origin, and public-destination policy before use.
- [ ] Dial-time DNS and every redirect are revalidated; DNS rebinding and mixed-address answers fail closed.
- [ ] Private PAR/token/revocation/PDS test listeners receive zero connections from malicious fixtures.
- [ ] No production code path in the OAuth/PDS boundary uses `http.DefaultClient` or an unguarded transport.
- [ ] Login failures use the standard error envelope and reveal no target/internal network detail.

### AV-017

- [ ] Every federated call has finite dial, TLS, header, idle, and total deadlines.
- [ ] Every success and error response is capped before decoding, with purpose-specific tested ceilings.
- [ ] Interactive and worker call sites apply explicit operation budgets and preserve cancellation.
- [ ] Timeout/oversize failures are retryable where appropriate and never misclassified as expired credentials.
- [ ] Focused integration tests, the database-backed suite, and race tests pass with no leaked goroutines or bodies.

## Dependencies and coordination

- **AV-013:** upgrade and pin Indigo/Go/dependencies, but do not remove this AppView-owned boundary after an upstream fix.
- **AV-014 / AV-015:** inbound admission and rate limiting reduce how often attackers can reach this outbound boundary; neither substitutes for it.
- **AV-007 / AV-009 / AV-019 / AV-021:** worker fencing, refresh serialization, callback compensation, and revocation jobs must all use the hardened clients.
- **AV-030:** centralize positive duration and deadline/lease validation.
- **AV-033 / AV-036:** make the network/security and race tests required CI gates before rollout.

## References

- [AppView OAuth BFF design](../superpowers/specs/2026-04-18-appview-oauth-bff-design.md)
- [CraftSky AT Protocol architecture reference](../../atproto-craft-social-app-reference.md)
- [AppView API architecture](../superpowers/specs/2026-04-21-appview-api-architecture-design.md)
- [AT Protocol OAuth security considerations](https://atproto.com/specs/oauth#security-considerations)
- [Go `net/http` Client documentation](https://pkg.go.dev/net/http#Client)
- [Go `net/http` Transport documentation](https://pkg.go.dev/net/http#Transport)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
