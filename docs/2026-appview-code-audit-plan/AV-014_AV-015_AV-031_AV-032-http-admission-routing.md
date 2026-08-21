# AV-014 / AV-015 / AV-031 / AV-032 — HTTP admission, abuse control, CORS, and routing contracts

- **Findings:** AV-014, “HTTP admission permits slow and large unauthenticated resource exhaustion”; AV-015, “Rate limiting is bypassable and grows memory and auth rows without bound”; AV-031, “Browser PATCH calls fail CORS preflight”; AV-032, “Unknown routes and method mismatches violate the JSON error contract”
- **Severity:** High / High / Medium / Medium
- **Priority/order:** Inbound boundary; land after the coordinated owner-lifecycle/auth foundation supplies explicit route access classes and before exposing AppView beyond a controlled development network
- **Status:** Planned
- **Source:** [AV-014](../2026-08-12-appview-code-audit.md#av-014--http-admission-permits-slow-and-large-unauthenticated-resource-exhaustion), [AV-015](../2026-08-12-appview-code-audit.md#av-015--rate-limiting-is-bypassable-and-grows-memory-and-auth-rows-without-bound), [AV-031](../2026-08-12-appview-code-audit.md#av-031--browser-patch-calls-fail-cors-preflight), and [AV-032](../2026-08-12-appview-code-audit.md#av-032--unknown-routes-and-method-mismatches-violate-the-json-error-contract)

## Shared implementation strategy

Turn `RoutePolicy` into the single source of truth for the `/v1/` admission pipeline, not merely a list checked beside manually registered handlers. The same compiled route catalogue should determine path/method matching, `Allow`, CORS preflight methods, authentication/member requirements, body policy, inner rate class, and observability route labels.

Apply admission in explicit cost order:

1. a bounded listener/active-connection gate plus `http.Server` header, idle, read, write, and header-size ceilings;
2. request ID, safe logging/metrics, panic recovery, and a global concurrent-request ceiling;
3. trusted-proxy-aware client-address derivation plus bounded global/IP rate limiting, before database authentication or body reads;
4. `ExpectedHost` rejection, still inside the outer budgets and request correlation;
5. `/v1/` path and method normalization, including contract-safe preflight handling;
6. a cheap `Content-Length`/content-type/body-presence precheck;
7. device, authentication, and current-member checks required by the route policy;
8. authenticated principal/device route-class limits; and
9. exactly one handler-owned, streaming, size- and time-bounded body decode.

This shared boundary prevents the four fixes from drifting: a CORS method cannot exist without a registered policy; wrong methods cannot fall into `ServeMux`’s text response; rate limiting cannot be placed behind an expensive body read; and an authenticated upload cannot be decoded twice. Because the app is pre-production, replace the old middleware composition and error behavior directly. Do not retain the current body-buffering or unbounded limiter as compatibility fallbacks.

## Finding closure

### AV-014 — HTTP admission permits slow and large unauthenticated resource exhaustion

The update closes AV-014 at both the connection and handler layers:

- Construct `http.Server` with non-zero `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, and `IdleTimeout`, plus a bounded `MaxHeaderBytes`. Set route-specific body-read deadlines after headers so ordinary JSON and large media uploads receive different, finite budgets.
- Bound accepted connections before `net/http` allocates a per-connection serve goroutine/header reader. Wrap the listener with a cancelable active-connection semaphore (or require and continuously verify an edge/proxy connection cap before direct reachability), hold a slot until `net.Conn.Close`, and stop accepting/close excess connections promptly when saturated. The post-header request semaphore remains a separate fairness/work bound; it cannot protect file descriptors or goroutines consumed by slow headers.
- Reject an over-limit positive `Content-Length` before authentication/body reads, but authenticate before streaming or allocating the payload. Chunked/unknown-length bodies are still capped while read.
- Replace `io.ReadAll` plus body rehydration in `middleware.BodyLimit` with `http.MaxBytesReader` (or a narrow equivalent) and one shared decode helper used by each handler. A handler receives one bounded stream and creates at most the representation it needs.
- Put a bounded in-flight/global admission semaphore before route work. A saturated process returns a short standard `503`/`429` response where headers have completed rather than admitting unbounded goroutines.
- Do not drain an unauthorized upload merely to keep the connection reusable. Close or stop reuse of that request connection according to the tested `net/http` behavior.

Connection-level failures can occur before a request ID or writable response exists, so a JSON envelope cannot be guaranteed for malformed/slow headers. Once the `/v1/` handler stack accepts headers, every generated rejection must use the standard camelCase envelope.

### AV-015 — Rate limiting is bypassable and grows memory and auth rows without bound

The update closes AV-015 with nested, bounded controls:

- Derive a canonical client address from `RemoteAddr`. Honor `Forwarded`/`X-Forwarded-For` only when the immediate peer is in configured trusted-proxy CIDRs, walking the chain from the trusted edge. Invalid or untrusted forwarding headers never replace the socket peer.
- Apply a global bucket and client-address/prefix bucket before authentication. Login additionally uses the device ID as a secondary key; rotating a caller-supplied device value cannot evade the IP/global limit.
- Keep per-OAuth-session/device limits after authentication. Invalid bearer floods hit the outer limiter before the auth database; valid principals receive route-class fairness.
- Replace the forever-growing bucket map with a capacity-bounded, TTL-evicted implementation behind a small `Limiter` interface. Define deterministic behavior at capacity: evict expired/oldest idle entries, and fail closed into a coarse overflow bucket instead of allocating another key.
- Run OAuth auth-request expiry independently on a scheduled worker. Bound pending-flow capacity transactionally, normalize `request_uri` into its own unique/indexed column, and make handoff recording address one row without a JSON-expression scan.

An in-process limiter is acceptable only while AppView is explicitly single-replica or protected by a separately verified shared edge limit. Before multiple replicas, supply a shared limiter implementation with the same decision contract; do not multiply the documented limit silently by replica count.

### AV-031 — Browser PATCH calls fail CORS preflight

The update closes AV-031 by deriving preflight behavior from the compiled route catalogue:

- For an allowed origin and known path, validate `Access-Control-Request-Method` against that path’s methods and return only the actual allowed method set plus `OPTIONS`. PATCH is therefore automatically advertised for every registered PATCH route.
- Validate requested headers against the explicit header allow-list. Add `Vary: Origin`, `Vary: Access-Control-Request-Method`, and `Vary: Access-Control-Request-Headers` so caches cannot reuse one preflight decision for another.
- Reject unknown paths, unsupported methods, disallowed origins, and disallowed headers consistently. Do not return a successful preflight merely because the request uses `OPTIONS`.
- Generate table-driven contract tests over every `V1RoutePolicies` entry so a future method addition cannot reproduce the mismatch.

Bearer auth remains header-based and CORS must not enable cookie credentials. `/oauth/*` protocol endpoints retain their separate AT Protocol/browser requirements.

### AV-032 — Unknown routes and method mismatches violate the JSON error contract

The update closes AV-032 by owning `/v1/` fallback and method negotiation before `ServeMux` can emit text:

- Parse and classify the escaped request path before `ServeMux`. Define the canonical API path as one leading slash, no repeated slash, no `.`/`..` segment (literal or percent-encoded after one strict decode), no encoded slash/backslash/NUL, and the exact catalogue trailing-slash shape. Reject non-canonical `/v1` paths with the JSON `404`/`400` contract; never let `ServeMux` emit its automatic clean-path/trailing-slash redirect. Do not redirect methods with bodies or reflect an attacker-controlled path in `Location`.
- Register or dispatch a method-neutral shadow for each known `/v1/` path pattern. Exact method-specific handlers win; any other method receives `405`, the canonical envelope, and an `Allow` header derived from the catalogue.
- Register a `/v1/` fallback that returns envelope-form `404`. Keep health and OAuth metadata/callback surfaces outside this API fallback.
- Preserve Go’s GET/HEAD semantics deliberately: either support HEAD for selected GET routes and test an empty response body, or reject it with a normalized `405`; do not inherit accidental `ServeMux` behavior.
- Run request-ID middleware before fallback/method handling so every application-generated error has a non-empty `requestId`.

## Desired outcome and invariants

- Slow headers, slow bodies, oversized headers/bodies, excessive concurrency, and excessive request rates all have finite resource costs.
- Active sockets and pre-header serve goroutines have a hard process/edge cap distinct from accepted-request concurrency.
- No unauthenticated request body is fully read or retained before the applicable cheap/global admission and authentication decisions.
- Each accepted body is bounded and decoded once.
- Caller-controlled device IDs and forwarding headers cannot bypass the outer abuse controls.
- Rate-limit memory and pending OAuth auth-request rows have explicit, testable upper bounds.
- The route catalogue and registered handlers are bijective: every policy has one handler and every `/v1/` handler has one policy.
- CORS, `Allow`, body policy, and middleware requirements come from that catalogue.
- Every application-generated non-2xx `/v1/*` response uses `{error, message, requestId}` with `application/json`; no internal target, token, raw header, or limiter key is exposed.
- Every public request, including an unexpected Host, passes through request correlation, global concurrency, and trusted-address/global/IP admission before `ExpectedHost`; Host rejection then occurs before route, CORS, authentication, or body work.

## Scope

### In scope

- `http.Server` construction and validated admission configuration.
- Global middleware ordering in `cmd/appview/server.go` and per-route composition in `internal/routes/routes.go`.
- A compiled route catalogue/registrar based on `internal/routes/policy.go`.
- Bounded body readers/decoders, concurrency control, trusted-proxy address parsing, nested limiters, and CORS.
- OAuth auth-request schema, capacity admission, expiry sweeper, and handoff lookup by normalized request URI.
- Contract, database, slow-client, concurrency, and browser-preflight tests.

### Out of scope

- Federated outbound HTTP safety, covered by AV-001 and AV-017.
- Replacing bearer authentication or changing the public/PDS versus private/AppView boundary.
- Generic OAuth middleware or any generic OAuth library.
- CDN/WAF vendor selection. General edge rate/WAF controls are defense in depth and do not replace in-process bounds; the sole alternative is an explicitly selected, verified hard connection-cap boundary that makes direct AppView reachability impossible and is tested as part of this plan.
- Individual handler business-logic authorization, except preserving the route policy’s auth/member requirements.

## Design decisions

1. **Compile and validate route policy at startup.** Duplicate method/path entries, policies without handlers, handlers without policies, invalid body/rate classes, and conflicting path patterns fail tests or startup.
2. **Use two rate-limit stages.** The outer stage knows only trusted network identity and global load; the inner stage adds authenticated session/device identity. Neither is a substitute for the other.
3. **Treat device IDs as hints, not security principals.** They improve fairness but never form the sole unauthenticated abuse key.
4. **Bound cardinality explicitly.** TTL alone is insufficient under a high-cardinality burst; every local cache has a hard entry cap and an overflow policy.
5. **Stream body limits.** `Content-Length` is a cheap early rejection hint, not trusted proof of size. The reader enforces the real cap for fixed and chunked bodies.
6. **Keep error ownership above `ServeMux`.** `/v1/` not-found and method-not-allowed responses are written by AppView’s envelope package.
7. **Derive CORS and `Allow`.** No hard-coded method string may drift from route registration.
8. **Prefer breaking cleanup to compatibility backfill.** Clear pre-production pending OAuth flows when adding a required normalized request URI and require login to restart.
9. **Reject non-canonical API paths explicitly.** Run one strict escaped-path validator before `ServeMux`; the catalogue's path spelling is authoritative and no automatic redirect is part of the `/v1` wire contract.

## Unified implementation plan

1. Add failure-first tests for slow headers/body, unauthenticated 15 MiB uploads, invalid-bearer floods, device-ID rotation, bucket cardinality, abandoned login rows, PATCH preflight, `/v1/` 404, and known-path 405 responses.
2. Extend `app.Config` with positive, bounded active-connection, server timeout/header/request-concurrency values; trusted-proxy CIDRs; outer/global/IP limit classes; limiter TTL/capacity; pending-auth capacity; and auth-request sweep interval/batch size. Reject invalid relationships at startup, coordinating with AV-030. If connection admission is delegated to an edge, startup/deployment verification must prove AppView is not directly reachable and the edge cap/timeouts are at least as strict; otherwise use the in-process listener cap.
3. Wrap the TCP listener in `cmd/appview/main.go` with a cancelable active-connection limiter whose token is acquired before returning from `Accept` and released exactly once by a wrapped connection's `Close`; on saturation, block accept with a bounded shutdown-aware wait or close newly accepted sockets without spawning handler goroutines. Then construct `http.Server` with explicit ceilings. Starting values to validate against current clients include a 5-second header timeout, 60-second idle timeout, and 32 KiB header cap. Derive the hard read and write ceilings from the longest permitted route budget: `WriteTimeout` must exceed the maximum body-read plus handler/downstream plus response-write budget by a safety margin, and no per-route body or outbound deadline may outlive it. Do not pair a longer permitted read with a shorter absolute write deadline. If route-specific response deadlines are used, document and test their interaction with the server ceiling.
4. Add a trusted-client-address parser using `net/netip`. Parse trusted proxy CIDRs at startup; ignore forwarding headers from other peers; normalize IPv4 and group IPv6 clients at a documented prefix for outer limiting while retaining a process-wide global bucket.
5. Define a concurrency gate and `Limiter` interface. Refactor `LocalRateLimiter` into a capacity-bounded TTL cache with amortized cleanup and a coarse overflow bucket. Avoid spawning cleanup goroutines per limiter/request; wire lifecycle to the server context.
6. Rebuild the outer `NewServer` order exactly as: listener connection cap/server header ceilings → request ID/recovery/safe logging → global request concurrency → trusted `RemoteAddr`/proxy resolution and global/IP rate limits → `ExpectedHost` → strict canonical escaped-path validation → route catalogue/CORS/authentication/body handling. The controls before `ExpectedHost` apply to all public traffic, so hostile hosts cannot bypass admission; request correlation makes `/v1/` Host/path rejection a canonical envelope. Metrics use bounded route/outcome labels even for unmatched paths.
7. Implement the strict `/v1` path validator outside `ServeMux`, then compile `V1RoutePolicies` into a registrar that pairs every policy with exactly one handler, computes allowed methods per exact canonical path pattern, and installs method-neutral 405 handlers plus `/v1/` 404 fallback. Remove `http.NotFoundHandler` for the API namespace and prevent `ServeMux` clean-path/trailing-slash redirects from owning API responses.
8. Refactor CORS to consume that method catalogue. Validate origin, requested method, and requested headers; derive method headers; add all required `Vary` fields; preserve the no-cookie-credentials contract.
9. Split body middleware into a non-reading admission check and handler-owned bounded decode helpers. Use `http.MaxBytesReader`, route-specific body-read deadlines, strict JSON decoding (single value, unknown-field policy where appropriate), and typed too-large/timeout errors. Delete the `ReadAll`/rehydration implementation and update route tests that assert body processing before auth.
10. Wire outer global/IP limits to all public traffic—including malformed/alternate Host—and inner route-class limits after auth/device middleware. Login requires global + client-address + device decisions before handle/DID discovery or PAR. Invalid bearer requests are blocked by outer limits before reaching `Authenticate` once the budget is exhausted.
11. Add an OAuth auth-request migration: remove pre-production pending rows, add a normalized `request_uri` column populated by `SaveAuthRequestInfo`, enforce the appropriate non-empty/unique constraint, and add the claim/expiry indexes used by handoff and sweeping. Replace `data->>'request_uri'` updates with an indexed, exactly-one-row operation.
12. Make pending OAuth flow creation a transactional capacity operation, serializing the count/insert safely (for example with a narrow PostgreSQL advisory lock or a counter row). Delete expired rows in the same bounded transaction before rejecting at capacity.
13. Add one context-owned auth-request sweeper that deletes expired rows in indexed batches. It must start once, stop on cancellation, use positive backoff, expose oldest-row/count/failure metrics, and not depend on callback traffic.
14. Update environment examples and operations docs with trusted-proxy and admission settings. Production startup must reject wildcard/ambiguous proxy trust and unsafe zero/negative limits; local development may explicitly trust only the Compose proxy network actually in use.
15. Run focused unit/HTTP/database tests, slow-client integration tests against a real `http.Server`, browser CORS tests, the full PostgreSQL suite, and race/static-analysis gates.

Likely files include `appview/cmd/appview/main.go`, `appview/cmd/appview/server.go`, `appview/internal/app/config.go`, `appview/internal/routes/routes.go`, `appview/internal/routes/policy.go`, `appview/internal/middleware/{body_limit,rate_limit,cors}.go`, `appview/internal/auth/store.go`, migrations, and their focused tests.

## Data, schema, migration, and reconciliation plan

Add a paired migration for normalized OAuth admission state:

- clear existing `oauth_auth_requests` because flows are short-lived and the application is pre-production;
- add `request_uri` as a normal constrained column written from `oauth.AuthRequestData.RequestURI` by `SaveAuthRequestInfo`;
- add a unique index for handoff lookup and retain/add an index that supports expiry batches by `created_at` (and a stable tie-breaker if batching requires one);
- add a capacity/counter row only if the selected transactional design requires it, with a constraint preventing negative or over-cap counts;
- update the authoritative test DDL and verify up/down/up migrations.

Coordinate this migration with AV-008/AV-018/AV-019 so the one-time handoff exchange and auth-request schema land in one coherent sequence. Do not create competing request-URI columns or cleanup workers.

There is no public-record/PDS migration. On rollout, invalidate pending auth starts and require users to start login again; existing completed OAuth/CraftSky sessions are unaffected unless the coordinated handoff migration intentionally resets them. Reconciliation consists of running the sweeper immediately, verifying pending rows are below capacity and have bounded age, and checking the unique lookup query plan.

## API, client, configuration, and operations impact

- Normal `/v1/*` success schemas remain unchanged. Errors become consistently parseable using the existing envelope.
- Known-path wrong methods return `405` with `Allow`; unknown `/v1/` paths return `404`; both include `requestId` and JSON content type.
- Correct browser PATCH preflights begin succeeding. Invalid origins/methods/headers no longer receive unconditional successful OPTIONS responses.
- Overload responses use stable `rate_limited` or `service_unavailable` errors and `Retry-After` where meaningful. Do not disclose the limiting key or whether an OAuth state exists.
- Login may return `429`/`503` before discovery when global, address, or pending-flow capacity is exhausted. Flutter should treat these as retryable and must not erase an existing session.
- Operators must configure trusted proxy CIDRs from actual deployment topology. Incorrect/missing trust means forwarding headers are ignored, not trusted broadly.
- Expose bounded counters/gauges for admission outcomes, in-flight requests, limiter entries/evictions/overflow, and pending/expired auth flows. Never label by IP, device ID, bearer, OAuth state, request URI, or arbitrary path.

## Security, failure, and race considerations

- Walk a forwarding chain from the known trusted peer; never select the leftmost header value blindly. Reject malformed address syntax and IPv4-mapped/private ambiguities.
- A global bucket protects against high-cardinality IP/device rotation; a hard map cap protects memory even when network identity is unbounded.
- Eviction must not reset an attacker into unlimited access. New keys at capacity share a conservative overflow bucket until space is reclaimed.
- Rate-limit checks should be atomic per request. If a shared backend is later used, define fail-open/fail-closed behavior per class; login and expensive unauthenticated work should fail closed with a bounded outage response.
- Pending-flow capacity check, expired-row cleanup, and insert must be serialized. Concurrent login starts cannot all observe one remaining slot.
- Handler timeouts/cancellation must reach database and outbound calls. Do not use an unbounded goroutine-based timeout wrapper that keeps work running after the response.
- A response that has begun streaming cannot be replaced by an error envelope. API handlers should validate/authenticate/decode before writing headers and keep their work inside the configured write budget.
- The listener token is a resource-safety primitive, not a user identity limit. Release it exactly once on every TLS/header failure, idle timeout, panic, normal keep-alive close, shutdown, and accept-loop error; never wait for a request semaphore while holding an unbounded set of accepted sockets.
- Decode paths once under a closed policy. Reject ambiguous escapes before routing, do not normalize backslashes or encoded separators, and do not send automatic redirects for `/v1` paths because redirects can alter method/body semantics and escape the canonical error envelope.
- CORS is a browser policy, not authentication or CSRF protection. Non-browser clients still pass through all normal auth/rate limits.

## Unified test plan

### Unit and contract tests

- Config rejects zero/negative/excessive timeouts, header/body sizes, capacities, windows, and invalid trusted-proxy CIDRs.
- Active-connection limiter covers acquire/release exactly once, canceled accept/shutdown, saturation, keep-alive reuse, TLS/header failure where applicable, and no token/file-descriptor leak.
- Trusted peer/untrusted peer, multiple proxies, malformed headers, IPv4-mapped IPv6, and IPv6-prefix normalization tables.
- Limiter window boundaries, TTL eviction, hard capacity, overflow-bucket behavior, concurrent checks, and bounded key count.
- Every policy has one handler and every registered `/v1/` handler has one policy; derived method sets include PATCH and exclude unregistered methods.
- Middleware-order tests prove alternate-Host traffic consumes global/IP/concurrency admission, returns a request-ID-bearing `/v1/` envelope, and never reaches route/auth/body handlers.
- Every policy method/path gets an allowed and disallowed-origin preflight case, including requested-header and `Vary` assertions.
- Unknown path and wrong method assert status, `Allow` where applicable, content type, exact envelope fields, and non-empty request ID.
- Canonical path table covers repeated leading/interior slash, `.`/`..`, percent-encoded dot/separator/backslash/NUL, invalid escapes, and required/extra trailing slash. Every rejected `/v1` form returns the chosen JSON 400/404 envelope with no `Location`; the handler and `ServeMux` redirect path are never reached.

### HTTP and database integration tests

- Real sockets saturate the active-connection cap with slow/incomplete headers, then attempt additional connections; file descriptors and serve goroutines stay within the declared bound, excess clients close/time out predictably, and slots recover after header timeout/client close/shutdown. Continue with slow fixed/chunked bodies, oversized headers, exactly-at-limit and one-byte-over bodies; resources are released within configured budgets.
- An unauthenticated/invalid-bearer upload proves the body decoder and large allocation are never invoked.
- A valid fixed-length and chunked upload is decoded once; memory does not contain duplicate full payload buffers.
- Invalid-bearer/device-rotation and alternate-Host floods are stopped by the outer global/address limits before repeated auth database/discovery or route/body work.
- Concurrent login starts cannot exceed pending capacity; expiry sweeping works without callbacks and uses the expected index.
- Normalized request-URI handoff updates exactly one row; missing/duplicate targets fail closed.

### Fault, concurrency, browser, and end-to-end tests

- Saturate pre-header connection capacity and post-header in-flight request capacity independently; assert bounded descriptors/goroutines, prompt bounded post-header responses, shutdown cancellation, recovery after release, and no token/goroutine leak.
- Run limiter, sweeper, and route tests under `go test -race` with clock/window and shutdown barriers.
- Exercise actual browser preflight/fetch for every PATCH surface from allowed and disallowed origins.
- Verify a reverse-proxy fixture supplies the expected client identity only when its CIDR is trusted.
- Confirm logs/metrics use stable route/reason labels and contain no IP, device, bearer, OAuth state, request URI, or unknown raw path.

## Per-ID traceability and acceptance criteria

### AV-014

- [ ] `http.Server` has tested non-zero header/read/write/idle and header-size ceilings.
- [ ] A listener or verified non-bypassable edge cap bounds active sockets/pre-header goroutines before `net/http` request handling; saturation, timeout, close, keep-alive, error, and shutdown tests prove exact slot recovery.
- [ ] No unauthenticated route reads or buffers an upload-sized body before cheap/global admission and required authentication.
- [ ] `BodyLimit` no longer retains and rehydrates a full payload; every body is stream-bounded and decoded once.
- [ ] Fixed-length, chunked, slow, and over-limit body tests terminate within finite resource budgets.
- [ ] Global in-flight admission has a hard cap and recovers cleanly after saturation.

### AV-015

- [ ] Login is limited by global and trusted client-address keys in addition to the caller-controlled device ID.
- [ ] Invalid bearer floods encounter an outer limiter before unbounded authentication database work.
- [ ] Limiter entry count remains at or below its configured cap under high-cardinality traffic, without eviction-based bypass.
- [ ] Pending OAuth flows have an atomic hard capacity and an independent indexed expiry sweeper.
- [ ] `request_uri` is normalized, unique/indexed, and no handoff update filters `data->>'request_uri'`.
- [ ] Multi-replica deployment is blocked or supplied a verified shared limiter/edge limit.

### AV-031

- [ ] PATCH is advertised and succeeds in preflight for every registered PATCH route and allowed origin.
- [ ] CORS allowed methods are generated from registered policies rather than a hand-maintained string.
- [ ] Unknown methods, disallowed origins, and disallowed headers do not receive successful preflight.
- [ ] `Origin`, requested method, and requested headers are all represented in `Vary` behavior.

### AV-032

- [ ] Every unknown `/v1/*` path returns a JSON `404` envelope with a non-empty request ID.
- [ ] Every known path with a wrong method returns a JSON `405` envelope and correct `Allow` header.
- [ ] Repeated-slash, dot-segment, escaped-separator/dot/NUL, invalid-escape, and trailing-slash variants are explicitly rejected before `ServeMux`, return the canonical envelope, and never redirect.
- [ ] GET/HEAD behavior is an explicit tested policy rather than an accidental `ServeMux` default.
- [ ] Health and `/oauth/*` routes retain their intended separate contracts.
- [ ] Route-catalogue completeness/uniqueness tests prevent future plain-text fallthroughs.

## Dependencies and coordination

- **AV-001 / AV-017:** the outer login limit reduces access to federated endpoints, while the outbound boundary remains responsible for SSRF/deadline/response-size safety.
- **AV-003:** adopt and test the lifecycle plan's explicit non-zero route access class while rebuilding registration; do not preserve the old `CurrentMemberRequired` Boolean/default.
- **AV-008 / AV-018 / AV-019:** coordinate one auth-request/handoff migration, atomic metadata recording, pending-session cleanup, and exchange-route admission.
- **AV-024:** safe bind/proxy defaults must agree with trusted-client-address configuration.
- **AV-030:** share positive-duration and relationship validation for server, limiter, sweep, and worker values.
- **AV-033 / AV-036:** make PostgreSQL, real-server, race, formatting, and static-analysis tests required gates.
- **AV-035:** use the same bounded-cache pattern for other process-lifetime throttle maps where semantics align.

## References

- [AppView API architecture](../superpowers/specs/2026-04-21-appview-api-architecture-design.md)
- [API wire alignment](../superpowers/specs/2026-04-22-api-wire-alignment-design.md)
- [AppView OAuth BFF design](../superpowers/specs/2026-04-18-appview-oauth-bff-design.md)
- [Go `net/http.Server` documentation](https://pkg.go.dev/net/http#Server)
- [Go `http.MaxBytesReader` documentation](https://pkg.go.dev/net/http#MaxBytesReader)
- [WHATWG Fetch CORS protocol](https://fetch.spec.whatwg.org/#http-cors-protocol)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
