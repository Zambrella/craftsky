# AppView API Architecture Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the cross-cutting API architecture defined in [`2026-04-21-appview-api-architecture-design.md`](../specs/2026-04-21-appview-api-architecture-design.md): `/v1/` version prefix, `X-Craftsky-Device-Id` header requirement, `{error, message, requestId}` error envelope, opaque-cursor pagination helpers, and the `last_device_id` column.

**Architecture:** This plan builds the shared infrastructure that every future v1 endpoint depends on. It does NOT add any v1 feature endpoint (feed, posts, profiles, etc.) — those land in their own specs + plans. The scope here is: one migration, one new `envelope` package, one new middleware, one context helpers extension, and the rerouting of the three existing auth/whoami routes under `/v1/`. OAuth routes and health routes stay at their current paths.

**Tech Stack:** Go 1.22+ (stdlib `net/http`, method/path routing), `pgx`, `golang-migrate/v4`, `slog`, existing `github.com/google/uuid` dependency. Tests run via `just test` against the compose Postgres.

---

## Background reading for the implementer

Read these before starting. They're short but load-bearing.

- [`docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`](../specs/2026-04-21-appview-api-architecture-design.md) — the spec this plan implements.
- [`docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md`](../specs/2026-04-18-appview-oauth-bff-design.md) — OAuth BFF that owns the auth routes this plan reroutes. §3.1 will need amending as part of this plan.
- [`AGENTS.md`](../../../AGENTS.md) — project rules. Coding conventions, Go toolchain, testing posture. This plan adds an "API conventions" subsection.
- [`appview/internal/routes/routes.go`](../../../appview/internal/routes/routes.go) — where route registration happens. Short file — read it in full.
- [`appview/internal/middleware/auth.go`](../../../appview/internal/middleware/auth.go) and [`logging.go`](../../../appview/internal/middleware/logging.go) — the patterns the new middleware follows.
- [`appview/internal/ctxkeys/ctxkeys.go`](../../../appview/internal/ctxkeys/ctxkeys.go) — the pattern the new `DeviceID` helpers follow.
- [`appview/migrations/`](../../../appview/migrations/) — existing migrations. Confirm highest number before writing the new one.

## Conventions this plan follows

- **TDD.** Every task writes the test first, confirms it fails, writes the minimal code to pass, confirms it passes, commits. This is rigid — don't batch.
- **One commit per task.** Tasks are small (a single file or a tightly-coupled pair). Frequent commits make reverts cheap.
- **`just test` is the only test runner.** Runs on the host, against compose Postgres via `localhost:5433`. Requires `just dev-d` running for integration tests. Pure unit tests (envelope, middleware) don't need Postgres but still run through `just test`.
- **`just fmt` after every non-trivial change.** Do not commit Go files that haven't been `gofmt`'d.
- **Naming.** snake_case for JSON fields, snake_case for error codes, UpperCamelCase for Go exports. This matches what the codebase already does.
- **No emojis in code or commit messages.**

## File structure

All paths are relative to repo root.

**New files:**

- `appview/migrations/000006_craftsky_sessions_device_id.up.sql` — adds `last_device_id TEXT` column.
- `appview/migrations/000006_craftsky_sessions_device_id.down.sql` — drops the column.
- `appview/internal/api/envelope/envelope.go` — `WriteError`, error-code constants, the JSON envelope struct.
- `appview/internal/api/envelope/envelope_test.go` — unit tests for the above.
- `appview/internal/api/envelope/cursor.go` — opaque cursor encode/decode helpers.
- `appview/internal/api/envelope/cursor_test.go` — unit tests for cursor helpers.
- `appview/internal/middleware/device_id.go` — middleware that validates `X-Craftsky-Device-Id` and injects it into context.
- `appview/internal/middleware/device_id_test.go` — unit tests for the middleware.

**Modified files:**

- `appview/internal/ctxkeys/ctxkeys.go` — add `DeviceIDKey`, `GetDeviceID`, `WithDeviceID`.
- `appview/internal/middleware/auth.go` — re-export `GetDeviceID`/`WithDeviceID` to mirror the existing DID helpers.
- `appview/internal/auth/craftsky_session.go` — new method `TouchDeviceID(ctx, token, deviceID)` that updates `last_device_id` opportunistically, throttled by the same window as `last_seen_at`.
- `appview/internal/auth/craftsky_session_test.go` — test the new method.
- `appview/internal/routes/routes.go` — reroute `/auth/login`, `/auth/logout`, `/whoami` under `/v1/`. Compose the new device-id middleware over `Authenticated` for authenticated routes.
- `appview/internal/routes/routes_test.go` — update existing tests to the new paths; add test for envelope shape on error; add test for device-id middleware composition.
- `AGENTS.md` — add an "API conventions" subsection inside "Coding Conventions."
- `docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md` — amend §3.1 to reflect the `/v1/` prefix on the Craftsky-internal auth endpoints.

## Chunk boundaries

- **Chunk 1:** Migration + schema addition. Purely additive DB change.
- **Chunk 2:** Envelope package (error writer + cursor helpers). Pure unit work.
- **Chunk 3:** Context helpers + middleware. Adds `DeviceID` plumbing.
- **Chunk 4:** Craftsky-session store integration for `last_device_id`.
- **Chunk 5:** Route rewiring + routes tests + cross-doc consistency updates.

---

## Chunk 1: Migration for `last_device_id`

### Task 1.1: Verify migration number is still `000006`

**Files:**
- Inspect: `appview/migrations/`

- [ ] **Step 1:** List existing migrations.

```bash
ls appview/migrations/
```

Expected: highest file prefix is `000005_drop_test_posts`. If a newer migration has landed, use the next free number for all subsequent tasks in this plan and note the change here.

If the number has changed, update every occurrence of `000006` in this plan before proceeding.

### Task 1.2: Write the up migration

**Files:**
- Create: `appview/migrations/000006_craftsky_sessions_device_id.up.sql`

- [ ] **Step 1:** Create the up migration.

```sql
-- Add per-device correlation column to craftsky_sessions.
-- See docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md §3.3.
ALTER TABLE craftsky_sessions
  ADD COLUMN last_device_id TEXT;
```

No index. We don't query by `last_device_id` in v1.

### Task 1.3: Write the down migration

**Files:**
- Create: `appview/migrations/000006_craftsky_sessions_device_id.down.sql`

- [ ] **Step 1:** Create the down migration.

```sql
ALTER TABLE craftsky_sessions
  DROP COLUMN last_device_id;
```

### Task 1.4: Apply the migration and verify

**Files:**
- Apply against dev Postgres.

- [ ] **Step 1:** Make sure compose is up.

```bash
just dev-d
```

- [ ] **Step 2:** Run the migration.

```bash
just migrate up
```

Expected output: shows the new migration being applied. No errors.

- [ ] **Step 3:** Verify the column exists.

```bash
just psql -c '\d craftsky_sessions'
```

Expected: a `last_device_id | text` row appears in the output.

- [ ] **Step 4:** Run the down migration and verify it reverts.

```bash
just migrate down 1
just psql -c '\d craftsky_sessions'
```

Expected: no `last_device_id` column.

- [ ] **Step 5:** Re-apply up so the DB is in the final state.

```bash
just migrate up
just psql -c '\d craftsky_sessions'
```

Expected: column is back.

### Task 1.5: Commit

- [ ] **Step 1:** Commit the migration.

```bash
git add appview/migrations/000006_craftsky_sessions_device_id.up.sql \
        appview/migrations/000006_craftsky_sessions_device_id.down.sql
git commit -m "$(cat <<'EOF'
feat(appview): add last_device_id column to craftsky_sessions

Tracks the most recent X-Craftsky-Device-Id seen per session.
Groundwork for the API architecture spec — no behaviour yet.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Chunk 2: Envelope package (errors + cursors)

### Task 2.1: Write failing test for `WriteError`

**Files:**
- Create: `appview/internal/api/envelope/envelope_test.go`

- [ ] **Step 1:** Write the test.

```go
package envelope_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/api/envelope"
)

func TestWriteError_WritesCanonicalShape(t *testing.T) {
	rec := httptest.NewRecorder()
	envelope.WriteError(rec, 422, "validation_failed", "bad input", "req-123", nil)

	if rec.Code != 422 {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if body["error"] != "validation_failed" {
		t.Errorf("error = %v, want validation_failed", body["error"])
	}
	if body["message"] != "bad input" {
		t.Errorf("message = %v, want bad input", body["message"])
	}
	if body["requestId"] != "req-123" {
		t.Errorf("requestId = %v, want req-123", body["requestId"])
	}
	if _, ok := body["fields"]; ok {
		t.Errorf("fields should be omitted when nil; got %v", body["fields"])
	}
}

func TestWriteError_IncludesFieldsWhenProvided(t *testing.T) {
	rec := httptest.NewRecorder()
	envelope.WriteError(rec, 422, "validation_failed", "bad input", "req-1", map[string]string{
		"text":     "exceeds max length",
		"material": "unknown code",
	})

	var body struct {
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if body.Fields["text"] != "exceeds max length" {
		t.Errorf("fields.text = %q", body.Fields["text"])
	}
	if body.Fields["material"] != "unknown code" {
		t.Errorf("fields.material = %q", body.Fields["material"])
	}
}
```

### Task 2.2: Run test, confirm it fails

- [ ] **Step 1:** Run.

```bash
just test
```

Expected: fails with `cannot find package "social.craftsky/appview/internal/api/envelope"` or similar.

### Task 2.3: Implement `envelope.go`

**Files:**
- Create: `appview/internal/api/envelope/envelope.go`

- [ ] **Step 1:** Implement.

```go
// Package envelope provides shared helpers for emitting the API's
// canonical JSON shapes.
//
// Every 4xx/5xx response produced by a v1 handler should go through
// WriteError. See docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md §6.
package envelope

import (
	"encoding/json"
	"net/http"
)

// Error is the JSON body shape for every non-2xx response.
type Error struct {
	Error     string            `json:"error"`
	Message   string            `json:"message"`
	RequestID string            `json:"requestId"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// WriteError serialises a canonical error response to w with the given
// HTTP status code. fields may be nil; it is omitted from the JSON when
// empty.
//
// requestID should be the per-request correlation ID (in this codebase:
// middleware.GetRunID(r.Context())). Pass "" only from tests or from
// code paths that run before the Logging middleware.
func WriteError(w http.ResponseWriter, status int, code, message, requestID string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Error{
		Error:     code,
		Message:   message,
		RequestID: requestID,
		Fields:    fields,
	})
}
```

### Task 2.4: Run test, confirm it passes

- [ ] **Step 1:** Run.

```bash
just test
```

Expected: PASS for both `TestWriteError_*`.

### Task 2.5: Commit envelope error helper

- [ ] **Step 1:** Commit.

```bash
just fmt
git add appview/internal/api/envelope/envelope.go appview/internal/api/envelope/envelope_test.go
git commit -m "$(cat <<'EOF'
feat(appview): add canonical API error envelope helper

envelope.WriteError emits the {error, message, requestId, fields?}
shape from the API architecture spec §6.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 2.6: Write failing test for cursor encode/decode

**Files:**
- Create: `appview/internal/api/envelope/cursor_test.go`

- [ ] **Step 1:** Write the test.

```go
package envelope_test

import (
	"testing"

	"social.craftsky/appview/internal/api/envelope"
)

func TestCursor_RoundTrip(t *testing.T) {
	in := map[string]any{
		"after": "2026-04-21T12:00:00Z",
		"id":    float64(42), // json numbers decode as float64
	}
	encoded, err := envelope.EncodeCursor(in)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if encoded == "" {
		t.Fatal("encoded cursor should not be empty")
	}

	out, err := envelope.DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["after"] != in["after"] {
		t.Errorf("after = %v, want %v", out["after"], in["after"])
	}
	if out["id"] != in["id"] {
		t.Errorf("id = %v, want %v", out["id"], in["id"])
	}
}

func TestCursor_EmptyStringDecodesToEmptyMap(t *testing.T) {
	out, err := envelope.DecodeCursor("")
	if err != nil {
		t.Fatalf("decode empty: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("empty cursor → non-empty map: %v", out)
	}
}

func TestCursor_MalformedReturnsInvalidCursorError(t *testing.T) {
	_, err := envelope.DecodeCursor("not-valid-base64url!!!")
	if err == nil {
		t.Fatal("expected error for malformed cursor")
	}
	if err != envelope.ErrInvalidCursor {
		t.Errorf("err = %v, want ErrInvalidCursor", err)
	}
}

func TestCursor_NonJSONPayloadReturnsInvalidCursorError(t *testing.T) {
	// A valid base64url that doesn't decode to JSON.
	bad := "bm90LWpzb24" // "not-json"
	_, err := envelope.DecodeCursor(bad)
	if err != envelope.ErrInvalidCursor {
		t.Errorf("err = %v, want ErrInvalidCursor", err)
	}
}
```

### Task 2.7: Run test, confirm it fails

- [ ] **Step 1:** Run.

```bash
just test
```

Expected: fails — `EncodeCursor`, `DecodeCursor`, `ErrInvalidCursor` undefined.

### Task 2.8: Implement `cursor.go`

**Files:**
- Create: `appview/internal/api/envelope/cursor.go`

- [ ] **Step 1:** Implement.

```go
package envelope

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// ErrInvalidCursor is returned by DecodeCursor when the input is not a
// valid base64url-encoded JSON object. Handlers should map this to a
// 400 with error code "invalid_cursor".
var ErrInvalidCursor = errors.New("invalid cursor")

// EncodeCursor serialises payload as base64url-encoded JSON. Handlers
// use it to produce the "cursor" field on paginated responses. The
// format is deliberately opaque — clients must not inspect it.
func EncodeCursor(payload map[string]any) (string, error) {
	if len(payload) == 0 {
		return "", nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeCursor is the inverse of EncodeCursor. An empty input returns
// an empty map (clients omit the cursor on the first page). Malformed
// input returns ErrInvalidCursor.
func DecodeCursor(cursor string) (map[string]any, error) {
	if cursor == "" {
		return map[string]any{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, ErrInvalidCursor
	}
	return out, nil
}
```

### Task 2.9: Run test, confirm it passes

- [ ] **Step 1:** Run.

```bash
just test
```

Expected: all four `TestCursor_*` tests pass.

### Task 2.10: Commit cursor helper

- [ ] **Step 1:** Commit.

```bash
just fmt
git add appview/internal/api/envelope/cursor.go appview/internal/api/envelope/cursor_test.go
git commit -m "$(cat <<'EOF'
feat(appview): add opaque cursor encode/decode helpers

Base64url-encoded JSON cursors for paginated list endpoints per the
API architecture spec §5. Malformed input surfaces a sentinel
ErrInvalidCursor that handlers map to 400 invalid_cursor.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Chunk 3: Device-ID context helpers + middleware

### Task 3.1: Add `DeviceID` to ctxkeys

**Files:**
- Modify: `appview/internal/ctxkeys/ctxkeys.go`

- [ ] **Step 1:** Append new key + helpers to the existing constants/functions.

At the end of the `const` block:

```go
const (
	DIDKey            contextKey = "did"
	OAuthSessionIDKey contextKey = "oauth_session_id"
	DeviceIDKey       contextKey = "device_id"
)
```

After the existing `WithOAuthSessionID` function, append:

```go
// GetDeviceID extracts the X-Craftsky-Device-Id injected by the
// DeviceID middleware. Returns ("", false) if not present.
func GetDeviceID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(DeviceIDKey).(string)
	return id, ok
}

// WithDeviceID stores id in ctx under DeviceIDKey.
func WithDeviceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, DeviceIDKey, id)
}
```

### Task 3.2: Write failing test for ctxkeys helpers

**Files:**
- Modify: `appview/internal/ctxkeys/ctxkeys_test.go` (if missing, create; likely doesn't exist — check `ls appview/internal/ctxkeys/`)

- [ ] **Step 1:** If `ctxkeys_test.go` doesn't exist, create it with this content; otherwise append these tests.

```go
package ctxkeys_test

import (
	"context"
	"testing"

	"social.craftsky/appview/internal/ctxkeys"
)

func TestDeviceID_RoundTrip(t *testing.T) {
	ctx := ctxkeys.WithDeviceID(context.Background(), "dev-abc")
	got, ok := ctxkeys.GetDeviceID(ctx)
	if !ok || got != "dev-abc" {
		t.Errorf("got (%q, %v), want (dev-abc, true)", got, ok)
	}
}

func TestDeviceID_AbsentReturnsFalse(t *testing.T) {
	_, ok := ctxkeys.GetDeviceID(context.Background())
	if ok {
		t.Error("ok should be false for empty context")
	}
}
```

### Task 3.3: Run tests, confirm they pass

- [ ] **Step 1:** Run.

```bash
just test
```

Expected: both `TestDeviceID_*` pass. (They were effectively test-after because the helpers are a trivial mirror of existing ones — keeping TDD discipline doesn't require writing these before the implementation, but confirm the tests exercise the exported surface correctly.)

### Task 3.4: Commit ctxkeys additions

- [ ] **Step 1:** Commit.

```bash
just fmt
git add appview/internal/ctxkeys/ctxkeys.go appview/internal/ctxkeys/ctxkeys_test.go
git commit -m "$(cat <<'EOF'
feat(appview): add DeviceID context key + helpers

Mirrors the existing DID / OAuth session-ID helpers so the upcoming
X-Craftsky-Device-Id middleware can inject values without cycles.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 3.5: Write failing test for device-id middleware

**Files:**
- Create: `appview/internal/middleware/device_id_test.go`

- [ ] **Step 1:** Write the test.

```go
package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeviceID_AcceptsValidHeaderAndInjectsCtx(t *testing.T) {
	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := GetDeviceID(r.Context())
		seen = id
		w.WriteHeader(http.StatusOK)
	})
	h := DeviceID(discardLogger())(next)

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Craftsky-Device-Id", "2c3f6a1e-0b4d-4cf5-9aa1-f0b4a9c9e1b3")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if seen != "2c3f6a1e-0b4d-4cf5-9aa1-f0b4a9c9e1b3" {
		t.Errorf("ctx device id = %q, want the sent value", seen)
	}
}

func TestDeviceID_MissingHeaderReturns400Envelope(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run")
	})
	h := DeviceID(discardLogger())(next)

	req := httptest.NewRequest("GET", "/x", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not json: %v", err)
	}
	if body["error"] != "missing_device_id" {
		t.Errorf("error = %v, want missing_device_id", body["error"])
	}
}

func TestDeviceID_EmptyHeaderReturns400(t *testing.T) {
	h := DeviceID(discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run")
	}))

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Craftsky-Device-Id", "")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestDeviceID_TooLongHeaderReturns400(t *testing.T) {
	h := DeviceID(discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run")
	}))

	long := make([]byte, 257)
	for i := range long {
		long[i] = 'a'
	}
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Craftsky-Device-Id", string(long))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
```

### Task 3.6: Run test, confirm it fails

- [ ] **Step 1:** Run.

```bash
just test
```

Expected: fails — `DeviceID`, `GetDeviceID` undefined in the `middleware` package.

### Task 3.7: Implement the middleware

**Files:**
- Create: `appview/internal/middleware/device_id.go`

- [ ] **Step 1:** Implement.

```go
package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

// maxDeviceIDLen bounds header size to prevent accidental / abusive
// client bugs from bloating the log stream. 256 bytes is comfortable
// headroom for UUIDs and ULIDs.
const maxDeviceIDLen = 256

// GetDeviceID extracts the X-Craftsky-Device-Id injected by DeviceID.
func GetDeviceID(ctx context.Context) (string, bool) {
	return ctxkeys.GetDeviceID(ctx)
}

// WithDeviceID stores id in ctx under the same key as the middleware.
// Exported for tests that want to skip middleware setup.
func WithDeviceID(ctx context.Context, id string) context.Context {
	return ctxkeys.WithDeviceID(ctx, id)
}

// DeviceID returns middleware that requires a non-empty
// X-Craftsky-Device-Id header and injects its value into the request
// context.
//
// Missing, empty, or over-length headers return 400 with the canonical
// error envelope. See:
// docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md §3.1
//
// Compose on top of Authenticated on any v1 route that requires auth:
//
//	h := Authenticated(svc, log)(DeviceID(log)(handler))
//
// The middleware does NOT verify that the device ID matches any
// persisted value; recording it on craftsky_sessions is the handler
// chain's responsibility.
func DeviceID(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Craftsky-Device-Id")
			if id == "" || len(id) > maxDeviceIDLen {
				logger.Warn("device-id: missing or invalid header",
					slog.Int("len", len(id)),
					slog.String("run_id", GetRunID(r.Context())))
				envelope.WriteError(w, http.StatusBadRequest,
					"missing_device_id",
					"X-Craftsky-Device-Id header is required",
					GetRunID(r.Context()),
					nil)
				return
			}
			ctx := ctxkeys.WithDeviceID(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

### Task 3.8: Run tests, confirm they pass

- [ ] **Step 1:** Run.

```bash
just test
```

Expected: all four `TestDeviceID_*` tests pass. The existing `TestAuthenticated_*` tests continue to pass.

### Task 3.9: Commit middleware

- [ ] **Step 1:** Commit.

```bash
just fmt
git add appview/internal/middleware/device_id.go appview/internal/middleware/device_id_test.go
git commit -m "$(cat <<'EOF'
feat(appview): add X-Craftsky-Device-Id middleware

Validates presence and bounded length, writes the canonical error
envelope on failure, injects the value into request context via
ctxkeys. Per API architecture spec §3.1.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Chunk 4: Persist `last_device_id` on the session

### Task 4.1: Extend the test harness DDL with `last_device_id`

The integration tests use `withAuthSchema(t)` in `appview/internal/auth/store_test.go`, which sets up a per-test private schema with hardcoded DDL for `oauth_sessions`, `oauth_auth_requests`, and `craftsky_sessions`. The production migration (Chunk 1) runs against the shared compose DB only — the per-test DDL must be updated separately or the new tests will fail with a missing-column error.

**Files:**
- Modify: `appview/internal/auth/store_test.go` (the `withAuthSchema` helper's DDL string, around line 65)

- [ ] **Step 1:** Add a `last_device_id TEXT` column to the `craftsky_sessions` DDL string inside `withAuthSchema`. Place it alongside `device_label`:

```go
CREATE TABLE ` + schema + `.craftsky_sessions (
    token_hash        BYTEA NOT NULL PRIMARY KEY,
    account_did       TEXT NOT NULL,
    oauth_session_id  TEXT NOT NULL,
    device_label      TEXT,
    last_device_id    TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at        TIMESTAMPTZ,
    FOREIGN KEY (account_did, oauth_session_id)
        REFERENCES ` + schema + `.oauth_sessions (account_did, session_id)
        ON DELETE CASCADE
);
```

- [ ] **Step 2:** Run `just test` and confirm existing tests still pass (no regression from the schema tweak).

### Task 4.2: Write failing test for `TouchDeviceID`

**Files:**
- Modify: `appview/internal/auth/craftsky_session_test.go`

Follow the existing pattern used by `TestCraftskySession_Lookup_HappyPath`: call `withAuthSchema(t)`, insert the FK row in `oauth_sessions` manually, then `store.Create(...)`.

- [ ] **Step 1:** Append two new test functions to the end of the file.

```go
func TestCraftskySession_TouchDeviceID_PersistsValue(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewCraftskySessionStore(pool, 0) // throttle disabled
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ('did:plc:a', 's1', '{}')`)
	if err != nil {
		t.Fatalf("seed oauth_sessions: %v", err)
	}
	token, err := store.Create(ctx, "did:plc:a", "s1", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.TouchDeviceID(ctx, token, "dev-xyz"); err != nil {
		t.Fatalf("TouchDeviceID: %v", err)
	}

	hash := sha256.Sum256([]byte(token))
	var got *string
	err = pool.QueryRow(ctx,
		`SELECT last_device_id FROM craftsky_sessions WHERE token_hash = $1`,
		hash[:]).Scan(&got)
	if err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if got == nil || *got != "dev-xyz" {
		t.Errorf("last_device_id = %v, want dev-xyz", got)
	}
}

func TestCraftskySession_TouchDeviceID_ThrottlesRepeats(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewCraftskySessionStore(pool, time.Hour)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ('did:plc:b', 's2', '{}')`)
	if err != nil {
		t.Fatalf("seed oauth_sessions: %v", err)
	}
	token, err := store.Create(ctx, "did:plc:b", "s2", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.TouchDeviceID(ctx, token, "dev-first"); err != nil {
		t.Fatalf("TouchDeviceID 1: %v", err)
	}
	if err := store.TouchDeviceID(ctx, token, "dev-second"); err != nil {
		t.Fatalf("TouchDeviceID 2: %v", err)
	}

	hash := sha256.Sum256([]byte(token))
	var got *string
	err = pool.QueryRow(ctx,
		`SELECT last_device_id FROM craftsky_sessions WHERE token_hash = $1`,
		hash[:]).Scan(&got)
	if err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if got == nil || *got != "dev-first" {
		t.Errorf("last_device_id = %v, want dev-first (throttle should have blocked the second write)", got)
	}
}
```

### Task 4.3: Run test, confirm it fails

- [ ] **Step 1:** Run.

```bash
just test
```

Expected: compile error — `store.TouchDeviceID` undefined.

### Task 4.4: Implement `TouchDeviceID`

**Files:**
- Modify: `appview/internal/auth/craftsky_session.go`

- [ ] **Step 1:** Add a second in-memory throttle map alongside `lastSeenMemory`, keyed identically but tracked independently so `TouchDeviceID` and `maybeTouchLastSeen` don't clobber each other's throttle state.

In the struct definition, add a sibling map:

```go
type CraftskySessionStore struct {
	pool             *pgxpool.Pool
	lastSeenThrottle time.Duration

	mu              sync.Mutex
	lastSeenMemory  map[string]time.Time
	deviceIDMemory  map[string]time.Time
}
```

Update `NewCraftskySessionStore`:

```go
func NewCraftskySessionStore(pool *pgxpool.Pool, lastSeenThrottle time.Duration) *CraftskySessionStore {
	return &CraftskySessionStore{
		pool:             pool,
		lastSeenThrottle: lastSeenThrottle,
		lastSeenMemory:   make(map[string]time.Time),
		deviceIDMemory:   make(map[string]time.Time),
	}
}
```

Add the method:

```go
// TouchDeviceID updates last_device_id on the craftsky_sessions row
// identified by token, at most once per lastSeenThrottle interval per
// token. It is safe to call on every authenticated request; the
// in-memory throttle bounds write load. Errors are returned to the
// caller but are generally non-fatal — the middleware fires this
// off-path.
func (s *CraftskySessionStore) TouchDeviceID(ctx context.Context, token, deviceID string) error {
	hash := sha256.Sum256([]byte(token))
	key := fmt.Sprintf("%x", hash)
	s.mu.Lock()
	last, ok := s.deviceIDMemory[key]
	now := time.Now()
	if ok && now.Sub(last) < s.lastSeenThrottle {
		s.mu.Unlock()
		return nil
	}
	s.deviceIDMemory[key] = now
	s.mu.Unlock()
	_, err := s.pool.Exec(ctx,
		`UPDATE craftsky_sessions SET last_device_id = $1 WHERE token_hash = $2`,
		deviceID, hash[:])
	return err
}
```

### Task 4.5: Run test, confirm it passes

- [ ] **Step 1:** Run.

```bash
just test
```

Expected: both `TestCraftskySession_TouchDeviceID_*` tests pass. No regressions in the existing store tests.

### Task 4.6: Commit store additions

- [ ] **Step 1:** Commit.

```bash
just fmt
git add appview/internal/auth/craftsky_session.go \
        appview/internal/auth/craftsky_session_test.go \
        appview/internal/auth/store_test.go
git commit -m "$(cat <<'EOF'
feat(appview): persist last_device_id on craftsky_sessions

New CraftskySessionStore.TouchDeviceID updates the column in the
background, throttled by the same window as last_seen_at. Also adds
the column to the per-test DDL in withAuthSchema so integration tests
can exercise the new code path. Wired up to the /v1 route tree in a
follow-up commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Chunk 5: Rewire routes under `/v1/` + cross-spec consistency

### Task 5.1: Write failing test for `/v1/` route paths

**Files:**
- Modify: `appview/internal/routes/routes_test.go`

- [ ] **Step 1:** Update the existing tests to hit the new paths, and add new tests for device-id enforcement and error envelope shape.

Replace the three existing test bodies with v1-prefixed paths. Add the required `X-Craftsky-Device-Id` header on authenticated requests. Keep the existing "unknown path returns 404" test at `/does-not-exist`.

```go
func TestAddRoutes_V1WhoAmIAuthenticatedReturnsDID(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	req.Header.Set("X-Dev-DID", "did:plc:from-header")
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "did:plc:from-header") {
		t.Errorf("body = %q, want containing 'did:plc:from-header'", rec.Body.String())
	}
}

func TestAddRoutes_V1WhoAmIWithoutAuthReturns401(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/whoami", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAddRoutes_V1WhoAmIWithoutDeviceIDReturns400(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rec.Body.String())
	}
}

func TestAddRoutes_LegacyUnprefixedWhoAmIReturns404(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/whoami", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (legacy path should be gone)", rec.Code)
	}
}

func TestAddRoutes_HealthStaysUnprefixed(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Errorf("status = 404; /health must stay unprefixed")
	}
}

func TestAddRoutes_OAuthClientMetadataStaysUnprefixed(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/oauth/client-metadata.json", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Errorf("status = 404; /oauth/client-metadata.json must stay unprefixed")
	}
}

func TestAddRoutes_UnknownPathReturns404(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
```

Delete the three old tests (`TestAddRoutes_WhoAmIAuthenticatedReturnsDID`, `TestAddRoutes_WhoAmIWithoutAuthReturns401`, and keep `TestAddRoutes_UnknownPathReturns404` unchanged).

### Task 5.2: Run tests, confirm the new v1 tests fail

- [ ] **Step 1:** Run.

```bash
just test
```

Expected: the new `TestAddRoutes_V1*` tests fail because the routes are still at the old unprefixed paths.

### Task 5.3: Reroute in `routes.go`

**Files:**
- Modify: `appview/internal/routes/routes.go`

- [ ] **Step 1:** Update the routes. Compose `DeviceID` on top of `Authenticated` for authenticated v1 routes. `/health`, `/healthz`, and `/oauth/*` stay at their current paths. Login is not authenticated but still requires the device-id header (so device attribution is available on the very first call).

```go
package routes

import (
	"context"
	"net/http"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
)

// AddRoutes registers all App View routes on mux.
func AddRoutes(ctx context.Context, mux *http.ServeMux, deps *app.Deps) {
	// Public ops.
	mux.Handle("GET /health", api.HealthHandler(deps.DB, deps.Logger))
	mux.Handle("GET /healthz", api.NewHealthHandler(deps.DB, deps.Consumer))

	// OAuth discovery endpoints (contracts with the AS; not versioned).
	oauthHandlers := auth.NewHTTPHandlers(
		deps.OAuthApp,
		deps.CraftskySessionStore,
		deps.DB,
		deps.Logger,
		deps.Config.Env == app.EnvDev,
	)
	mux.Handle("GET /oauth/client-metadata.json", oauthHandlers.ClientMetadataHandler())
	mux.Handle("GET /oauth/jwks.json", oauthHandlers.JWKSHandler())
	mux.Handle("GET /oauth/callback", oauthHandlers.CallbackHandler())

	// Middleware stacks.
	authN := middleware.Authenticated(deps.AuthService, deps.Logger)
	deviceID := middleware.DeviceID(deps.Logger)

	// v1 — unauthenticated but device-id required.
	mux.Handle("POST /v1/auth/login", deviceID(oauthHandlers.LoginHandler()))

	// v1 — authenticated + device-id required.
	mux.Handle("GET /v1/whoami", authN(deviceID(api.WhoAmIHandler())))
	mux.Handle("POST /v1/auth/logout", authN(deviceID(oauthHandlers.LogoutHandler())))

	// Fallthrough.
	mux.Handle("/", http.NotFoundHandler())
}
```

### Task 5.4: Run tests, confirm they pass

- [ ] **Step 1:** Run.

```bash
just test
```

Expected: all routes tests pass. All middleware tests pass. All auth tests pass.

- [ ] **Step 2:** Manually smoke-test in the running compose stack.

```bash
curl -i http://localhost:8080/v1/whoami
# → 401 (no Authorization header)

curl -i -H "X-Craftsky-Device-Id: dev-test" http://localhost:8080/v1/whoami
# → 401 (no Authorization header; device-id alone isn't enough)

curl -i -H "Authorization: Bearer anything" \
     -H "X-Dev-DID: did:plc:curl" \
     -H "X-Craftsky-Device-Id: dev-test" \
     http://localhost:8080/v1/whoami
# → 200 {"did":"did:plc:curl"}   (dev mode only)

curl -i http://localhost:8080/whoami  # legacy path → 404
```

Record any discrepancy and fix before continuing.

### Task 5.5: Commit route rewiring

- [ ] **Step 1:** Commit.

```bash
just fmt
git add appview/internal/routes/routes.go appview/internal/routes/routes_test.go
git commit -m "$(cat <<'EOF'
refactor(appview): move Craftsky API routes under /v1/

- Reroutes /whoami, /auth/login, /auth/logout to /v1/*.
- Requires X-Craftsky-Device-Id on every v1 request (including login),
  via the new DeviceID middleware composed over Authenticated.
- Keeps /health, /healthz, /oauth/* at their existing paths — those are
  ops / atproto-spec surfaces that aren't versioned here.

Per API architecture spec §2.1, §3.1.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 5.6: Amend OAuth BFF spec §3.1

**Files:**
- Modify: `docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md`

- [ ] **Step 1:** Read the current §3.1 table. It lists:

```
| POST | /auth/login | Craftsky client (Flutter/CLI) | … |
| POST | /auth/logout | Craftsky client (Flutter/CLI) | … |
```

Change the `Path` values to `/v1/auth/login` and `/v1/auth/logout`. Leave OAuth spec paths (`/oauth/*`) unchanged.

- [ ] **Step 2:** Add an errata note at the top of §3.1 (or at the top of the spec, whichever lands cleaner):

```markdown
> **Errata (2026-04-21):** The Craftsky-internal auth endpoints
> moved under `/v1/` as part of the API architecture spec
> ([`2026-04-21-appview-api-architecture-design.md`](./2026-04-21-appview-api-architecture-design.md)).
> OAuth AS-facing endpoints (`/oauth/*`) are unchanged.
```

### Task 5.7: Add AGENTS.md API conventions subsection

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1:** Inside the existing "Coding Conventions" section, add a new bullet (alongside the Go / Dart / SQL / Commits bullets):

```markdown
- **API:** The HTTP surface between the Flutter app and the AppView is governed by the API architecture spec ([`docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`](docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md)). Before adding or changing any route, read it — it fixes the `/v1/` prefix, auth headers, error envelope (`{error, message, requestId}`), opaque-cursor pagination, and URL conventions.
```

### Task 5.8: Commit spec amendment + AGENTS.md update

- [ ] **Step 1:** Commit.

```bash
git add docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md AGENTS.md
git commit -m "$(cat <<'EOF'
docs: reflect /v1/ prefix in OAuth BFF spec and AGENTS.md

- Update OAuth BFF spec §3.1 endpoint table to use /v1/ paths for the
  Craftsky-internal auth endpoints; add an errata note.
- Add an "API" bullet to AGENTS.md Coding Conventions pointing at the
  API architecture spec.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 5.9: Update the roadmap

**Files:**
- Modify: `docs/roadmap.md`

- [ ] **Step 1:** Mark the API architecture item under "v1 → AppView / API" as done (change `[ ]` to `[x]`) for the entry already linked to this spec.

### Task 5.10: Commit roadmap update

- [ ] **Step 1:** Commit.

```bash
git add docs/roadmap.md
git commit -m "$(cat <<'EOF'
docs: mark API architecture item done in roadmap

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Final verification

### Task 6.1: Full test run + fmt + vet

- [ ] **Step 1:** Run the full suite one more time.

```bash
just fmt
just test
```

Expected: all tests pass. No gofmt / vet output.

### Task 6.2: Smoke test the full API path in the running stack

- [ ] **Step 1:** Confirm compose is up.

```bash
just dev-d
```

- [ ] **Step 2:** Exercise every new behaviour.

```bash
# v1 path with all required headers → 200
curl -s -o /dev/null -w "%{http_code}\n" \
  -H "Authorization: Bearer anything" \
  -H "X-Dev-DID: did:plc:smoke" \
  -H "X-Craftsky-Device-Id: dev-smoke" \
  http://localhost:8080/v1/whoami
# Expected: 200

# v1 path without device-id → 400 with JSON envelope
curl -s \
  -H "Authorization: Bearer anything" \
  -H "X-Dev-DID: did:plc:smoke" \
  http://localhost:8080/v1/whoami
# Expected: JSON containing "missing_device_id"

# legacy /whoami → 404
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/whoami
# Expected: 404

# /health unchanged
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/health
# Expected: 200

# /oauth/client-metadata.json unchanged
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/oauth/client-metadata.json
# Expected: 200 (or whatever the existing handler returns — should NOT be 404)
```

- [ ] **Step 3:** If anything diverges, fix before calling the plan done.

### Task 6.3: Final consistency sweep

- [ ] **Step 1:** Skim the commit log for the branch.

```bash
git log --oneline main..HEAD
```

Each commit should be self-contained and reversible. If two commits are stepping on each other, consider rebasing locally (not force-pushing).

- [ ] **Step 2:** Confirm every file listed in the "File structure" section actually exists (or was modified).

```bash
ls appview/migrations/000006_craftsky_sessions_device_id.*.sql
ls appview/internal/api/envelope/
ls appview/internal/middleware/device_id*.go
```

- [ ] **Step 3:** Verify invariants the spec promised:
  - `/v1/whoami` responds with envelope shape on 400/401.
  - `/health` and `/oauth/*` are NOT under `/v1/`.
  - `last_device_id` column exists on `craftsky_sessions`.
  - No tests were left skipped or disabled.

### Task 6.4: Hand-off summary

- [ ] **Step 1:** Report completion to the user with:
  - Migration number used (confirm it wasn't bumped past `000006`).
  - Count of new tests added.
  - The URLs of the two amended docs (OAuth BFF spec + AGENTS.md).
  - Any spec ambiguity discovered during implementation that should feed back into the spec or roadmap.

---

## What this plan deliberately does NOT do

Cross-reference with the spec's "Non-goals" (§2) and "Future work" (§9):

- No feature endpoints (feed, profiles, posts, notifications). Each has its own spec + plan.
- No write-proxy infrastructure (DPoP-signing path from handler → user's PDS). Own spec.
- No rate limiting, no CORS, no request-body size limits, no observability rewrite. Listed in the roadmap.
- No OpenAPI document. Listed in the roadmap.
- No success-response envelope. The spec explicitly leaves responses bare for v1.
- No blob upload.
- No `/xrpc/*` pass-throughs.

If you find yourself reaching for one of these while executing the plan, stop. It's a separate spec. Note it in the roadmap and move on.
