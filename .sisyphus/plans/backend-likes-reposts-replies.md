# Backend Likes, Reposts, and Replies

## TL;DR
> **Summary**: Add full AppView backend support for liking, reposting, and replying to Craftsky posts, using existing AT Protocol lexicons and existing `/v1/*` API conventions. Likes/reposts become indexed interaction records with PDS write/delete endpoints, post responses gain engagement counts + viewer state, and replies gain shallow + full-thread read APIs.
> **Deliverables**:
> - Postgres migrations for like/repost interaction indexes and query support.
> - Firehose indexers for `social.craftsky.feed.like` and `social.craftsky.feed.repost`.
> - PDS-backed like/unlike and repost/unrepost HTTP handlers.
> - Engagement counts and viewer state on post responses/lists.
> - Direct-replies and nested-thread read endpoints.
> - Backend tests and agent-executed verification evidence.
> **Effort**: Large
> **Parallel**: YES - 3 waves
> **Critical Path**: Task 1 → Tasks 2-4 → Tasks 5-8 → Final Verification Wave

## Context
### Original Request
The user said: "Now that we have some basic post functionality I'd like to add the ability to like, repost and reply. Let's plan this out starting with the backend."

### Interview Summary
- Backend scope: full interaction backend.
- Repost model: pure repost only; quote posts stay on the existing `social.craftsky.feed.post` quote-embed path.
- Reply reads: include both direct replies and full nested thread reads.
- Engagement reads: expose counts and current-viewer state only; do not expose actor-list endpoints for likers/reposters.
- Test strategy: tests-after.

### Metis Review (gaps addressed)
- Exact HTTP contract fixed in this plan instead of leaving it to implementer judgment.
- Write consistency model is explicit: write endpoints return optimistic PDS-write success; AppView counts/state update when firehose/indexing catches up, except the immediate response may include the created/deleted record identity.
- Subject eligibility is explicit: write endpoints reject targets not present in `craftsky_posts`; indexers ignore interactions whose subject is not an indexed Craftsky post.
- Duplicate semantics are explicit: only one active like and one active repost count per actor+subject, even if multiple records exist on a PDS.
- Thread depth/caps are explicit to avoid unbounded recursive responses.

## Work Objectives
### Core Objective
Backend users can create/delete likes and reposts through the AppView, create replies using the existing post write path, and read engagement/reply/thread data from the AppView without the Flutter app reading Craftsky records directly from a PDS.

### Deliverables
- `craftsky_likes` and `craftsky_reposts` tables with idempotent active-state semantics.
- Like/repost indexers registered in the dispatcher.
- Interaction store methods used by handlers and response builders.
- Authenticated `/v1/posts/{did}/{rkey}/likes` and `/v1/posts/{did}/{rkey}/reposts` write/delete routes.
- `likeCount`, `repostCount`, `replyCount`, `viewerHasLiked`, `viewerHasReposted` on `PostResponse` and every endpoint returning posts.
- `GET /v1/posts/{did}/{rkey}/replies` for direct replies.
- `GET /v1/posts/{did}/{rkey}/thread` for nested thread reads.
- Handler/store/indexer tests plus final `just test` verification.

### Definition of Done (verifiable conditions with commands)
- `just dev-d` starts compose dependencies.
- `just test` passes from repo root.
- `POST /v1/posts/{did}/{rkey}/likes` writes `social.craftsky.feed.like` to caller PDS and returns `201` with `{uri,cid,rkey,subject,createdAt}`.
- `DELETE /v1/posts/{did}/{rkey}/likes` is idempotent and returns `204`.
- `POST /v1/posts/{did}/{rkey}/reposts` writes `social.craftsky.feed.repost` to caller PDS and returns `201` with `{uri,cid,rkey,subject,createdAt}`.
- `DELETE /v1/posts/{did}/{rkey}/reposts` is idempotent and returns `204`.
- Post read/list responses include counts + viewer booleans with camelCase keys.
- Direct replies and thread endpoints return deterministic, paginated/capped data without human intervention.

### Must Have
- Writes go through the user's PDS; reads come from AppView Postgres.
- Flutter/client-facing JSON remains camelCase.
- Use existing lexicons; do not change `lexicon/` unless a test proves generated types are stale.
- Use generated lexicon Go types for like/repost records; do not hand-roll record structs inside indexers.
- Preserve existing post reply write support in `POST /v1/posts`; do not create a separate reply record NSID.
- Indexers must be idempotent on repeated firehose events.
- One active like/repost per `(actor_did, subject_uri)` in AppView counts/state.

### Must NOT Have (guardrails, AI slop patterns, scope boundaries)
- Do not add actor-list endpoints (`likedBy`, `repostedBy`) in this backend phase.
- Do not add quote-repost APIs or change quote embed semantics.
- Do not add ranking, notifications, activity feeds, moderation expansion, or client UI work.
- Do not bypass AppView by making Flutter read Craftsky data directly from a PDS.
- Do not store PDS tokens on the device.
- Do not add SQL ORMs; use raw SQL/pgx patterns already present.
- Do not introduce `sqlc` unless the repository has already adopted it for these queries during a separate task; current `appview/queries/.gitkeep` indicates no query files yet.

## Verification Strategy
> ZERO HUMAN INTERVENTION - all verification is agent-executed.
- Test decision: tests-after using Go stdlib tests, handler fakes, and real Postgres integration tests.
- QA policy: Every task has agent-executed scenarios.
- Evidence: `.sisyphus/evidence/task-{N}-{slug}.{ext}`
- Final command: from repo root, run `just dev-d` if compose is not already running, then `just test`.

## Execution Strategy
### Parallel Execution Waves
> Target: 5-8 tasks per wave. <3 per wave (except final) = under-splitting.
> Extract shared dependencies as Wave-1 tasks for max parallelism.

Wave 1: Task 1 foundation schema/contracts.
Wave 2: Tasks 2-4 can run after Task 1: indexers, interaction store, response augmentation.
Wave 3: Tasks 5-8 can run after their dependencies: write/delete handlers, reply reads, thread reads, route/test integration.

### Dependency Matrix (full, all tasks)
- Task 1 blocks Tasks 2, 3, 4, 5, 6, 7, 8.
- Task 2 blocks final end-to-end indexer verification and contributes to Tasks 4-6.
- Task 3 blocks Tasks 4, 5, 6, 7, 8.
- Task 4 blocks response assertions in Tasks 5, 7, 8.
- Task 5 and Task 6 are independent after Tasks 1-4.
- Task 7 and Task 8 depend on Task 3 and existing post reply indexing.
- Final Verification Wave depends on Tasks 1-8.

### Agent Dispatch Summary (wave → task count → categories)
- Wave 1 → 1 task → unspecified-high.
- Wave 2 → 3 tasks → unspecified-high, quick, unspecified-high.
- Wave 3 → 4 tasks → unspecified-high, unspecified-high, unspecified-high, quick.
- Final Verification Wave → 4 review tasks → oracle, unspecified-high, unspecified-high, deep.

## TODOs
> Implementation + Test = ONE task. Never separate.
> EVERY task MUST have: Agent Profile + Parallelization + QA Scenarios.

- [x] 1. Add interaction storage migration and test fixtures

  **What to do**: Add `appview/migrations/000011_craftsky_interactions.up.sql` and `.down.sql`. Create `craftsky_likes` and `craftsky_reposts` with columns: `uri TEXT PRIMARY KEY`, `did TEXT NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE`, `rkey TEXT NOT NULL`, `cid TEXT NOT NULL`, `subject_uri TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE`, `subject_cid TEXT NOT NULL`, `record JSONB NOT NULL`, `created_at TIMESTAMPTZ NOT NULL`, `indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()`, `deleted_at TIMESTAMPTZ`. Add `UNIQUE (did, rkey)` and partial unique indexes enforcing one active row per `(did, subject_uri)` where `deleted_at IS NULL`. Add read indexes for `(subject_uri) WHERE deleted_at IS NULL`, `(did, subject_uri) WHERE deleted_at IS NULL`, and `(indexed_at DESC)`. Down migration drops reposts first, then likes. Update test inline schemas in affected `*_test.go` files only where tests use hand-written DDL.

  **Must NOT do**: Do not alter existing `craftsky_posts` columns. Do not create aggregate counter columns on `craftsky_posts`; counts must be computed from active interaction rows to avoid race-prone denormalization in this pass.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: schema decisions affect indexers, handlers, and query semantics.
  - Skills: [] - No external skill needed because lexicon files are not changed.
  - Omitted: [`atproto-lexicon`] - Not editing `lexicon/`.

  **Parallelization**: Can Parallel: NO | Wave 1 | Blocks: Tasks 2-8 | Blocked By: none

  **References** (executor has NO interview context - be exhaustive):
  - Pattern: `appview/migrations/000010_craftsky_posts.up.sql:3` - existing post table style, text identifiers, `JSONB record`, timestamps, and indexes.
  - Pattern: `appview/migrations/000010_craftsky_posts.down.sql:2` - simple down migration style.
  - Pattern: `appview/internal/testdb/testdb.go` - real-Postgres test helper referenced by existing integration tests.
  - Test: `appview/internal/index/craftsky_post_test.go` - inline DDL pattern that may need interaction tables for indexer tests.
  - Test: `appview/internal/api/post_store_test.go` - inline schema + seed helper pattern for store tests.

  **Acceptance Criteria** (agent-executable only):
  - [ ] `ls appview/migrations/000011_craftsky_interactions.up.sql appview/migrations/000011_craftsky_interactions.down.sql` succeeds.
  - [ ] A SQL inspection confirms both tables have partial unique active `(did, subject_uri)` indexes.
  - [ ] Running `just test` after later tasks does not fail due to missing interaction tables in tests.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Migration applies cleanly
    Tool: Bash
    Steps: Run `just dev-d`; run `just migrate up`.
    Expected: Migration command exits 0 and `craftsky_likes`/`craftsky_reposts` exist in Postgres.
    Evidence: .sisyphus/evidence/task-1-interaction-migration.txt

  Scenario: Duplicate active interaction rejected
    Tool: Bash
    Steps: Use `just psql -c` to insert one active like row, then attempt a second active like row for the same did+subject_uri.
    Expected: Second insert fails with unique-constraint violation; after setting `deleted_at`, a new active row for same did+subject_uri can be inserted.
    Evidence: .sisyphus/evidence/task-1-duplicate-active-like.txt
  ```

  **Commit**: YES | Message: `feat(appview): add interaction storage` | Files: `appview/migrations/000011_craftsky_interactions.*.sql`, affected test fixture files

- [x] 2. Add like and repost firehose indexers

  **What to do**: Add indexers under `appview/internal/index/` for `social.craftsky.feed.like` and `social.craftsky.feed.repost`. Each handles `create`, `update`, and `delete`: create/update unmarshals generated `craftskylex.FeedLike` or `craftskylex.FeedRepost`, parses `createdAt`, requires `Subject` non-nil with non-empty URI/CID, verifies actor is a Craftsky member, verifies `subject_uri` exists in `craftsky_posts`, soft-deletes any other active row for the same `(did, subject_uri)`, then upserts the current interaction row and clears `deleted_at`. Delete uses soft-delete (`deleted_at = now()`) so duplicate historical records can be audited while active counts stay correct. Register both indexers in `appview/internal/app/deps.go`.

  **Must NOT do**: Do not count interactions for missing/non-Craftsky subjects. Do not fail the whole tap consumer for non-member actors or missing subject posts; silently ignore these like the post indexer ignores non-members.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: firehose idempotency and record semantics need care.
  - Skills: [] - Generated types already exist.
  - Omitted: [`atproto-lexicon`] - No schema edits.

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: final verification | Blocked By: Task 1

  **References** (executor has NO interview context - be exhaustive):
  - Pattern: `appview/internal/index/craftsky_post.go:40` - NSID constant and collection filtering.
  - Pattern: `appview/internal/index/craftsky_post.go:56` - member gate, generated type unmarshal, timestamp parse, upsert.
  - Pattern: `appview/internal/index/craftsky_post.go:118` - idempotent `ON CONFLICT` upsert style.
  - Pattern: `appview/internal/index/craftsky_post.go:158` - delete handling style.
  - Pattern: `appview/internal/index/dispatcher.go:40` - dispatcher registration semantics.
  - Wiring: `appview/internal/app/deps.go:118` and `appview/internal/app/deps.go:124` - dispatcher setup and post indexer registration.
  - API/Type: `appview/internal/lexicon/craftsky/feedlike.go:16` - generated `FeedLike` with `Subject` strongRef.
  - API/Type: `appview/internal/lexicon/craftsky/feedrepost.go:16` - generated `FeedRepost` with `Subject` strongRef.

  **Acceptance Criteria** (agent-executable only):
  - [ ] New indexer tests cover create, duplicate delivery idempotency, delete/soft-delete, non-member ignored, and missing subject ignored for likes and reposts.
  - [ ] Dispatcher registration test or route wiring test proves both NSIDs are registered.
  - [ ] `go test ./internal/index ./internal/app` from `appview/` passes with `TEST_DATABASE_URL` set.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Like indexer materializes active state
    Tool: Bash
    Steps: Run targeted Go test for the new like indexer create/idempotency case.
    Expected: Exactly one active row exists for did+subject_uri after duplicate create events.
    Evidence: .sisyphus/evidence/task-2-like-indexer.txt

  Scenario: Missing subject does not inflate counts
    Tool: Bash
    Steps: Run targeted Go test for like/repost event whose subject_uri is absent from craftsky_posts.
    Expected: Test passes and zero interaction rows are inserted.
    Evidence: .sisyphus/evidence/task-2-missing-subject.txt
  ```

  **Commit**: YES | Message: `feat(appview): index post interactions` | Files: `appview/internal/index/*like*`, `appview/internal/index/*repost*`, `appview/internal/app/deps.go`, tests

- [x] 3. Add interaction store methods and shared wire types

  **What to do**: Add an interaction store in `appview/internal/api/` or extend `PostStore` if that yields less duplication. Required methods: resolve target post by `(did,rkey)` to `uri,cid`; find active like/repost by caller DID + subject URI for idempotent delete; count active likes/reposts for one or many post URIs; compute viewer booleans for one or many post URIs; count direct replies by parent URI; list direct replies with opaque cursor; load thread candidate rows by root URI with a fixed cap. Add request/response wire types for interaction write responses: `uri`, `cid`, `rkey`, `subject`, `createdAt`. Keep all JSON camelCase.

  **Must NOT do**: Do not add likers/reposters actor-list methods. Do not return PDS tokens or OAuth internals in any response type.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: query design feeds multiple handlers and response builders.
  - Skills: [] - Straight Go/Postgres work.
  - Omitted: [`flutter-use-http-package`] - Backend only.

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: Tasks 4-8 | Blocked By: Task 1

  **References** (executor has NO interview context - be exhaustive):
  - Pattern: `appview/internal/api/post_store.go:53` - store interface split for handler fakes.
  - Pattern: `appview/internal/api/post_store.go:70` - shared select column constant.
  - Pattern: `appview/internal/api/post_store.go:90` - `ReadOne` error mapping.
  - Pattern: `appview/internal/api/post_store.go:111` - cursor decoding and paginated list style.
  - Pattern: `appview/internal/api/envelope/` - opaque cursor helpers and error envelope helpers.
  - Test: `appview/internal/api/post_store_test.go` - real-Postgres store test style.

  **Acceptance Criteria** (agent-executable only):
  - [ ] Store tests prove counts ignore rows with `deleted_at IS NOT NULL`.
  - [ ] Store tests prove viewer booleans are true only for active rows by the current DID.
  - [ ] Store tests prove direct reply listing paginates with an opaque cursor.
  - [ ] Store tests prove thread loading obeys cap/depth inputs.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Engagement summary query
    Tool: Bash
    Steps: Run targeted Go test seeding two posts, active/deleted likes, active reposts, and current-viewer interactions.
    Expected: Counts and viewer booleans match seeded active rows only.
    Evidence: .sisyphus/evidence/task-3-engagement-summary.txt

  Scenario: Direct replies pagination
    Tool: Bash
    Steps: Run targeted Go test seeding three direct replies and requesting limit=2.
    Expected: First page has two replies and non-empty cursor; second page has remaining reply and empty cursor.
    Evidence: .sisyphus/evidence/task-3-reply-pagination.txt
  ```

  **Commit**: YES | Message: `feat(appview): add interaction queries` | Files: `appview/internal/api/*interaction*`, `appview/internal/api/post_store.go`, tests

- [x] 4. Add engagement fields to post responses

  **What to do**: Extend `PostResponse` with `LikeCount int`, `RepostCount int`, `ReplyCount int`, `ViewerHasLiked bool`, and `ViewerHasReposted bool` using JSON keys `likeCount`, `repostCount`, `replyCount`, `viewerHasLiked`, `viewerHasReposted`. Update `BuildPostResponse` or add a companion builder so `GET /v1/posts/{did}/{rkey}`, `GET /v1/profiles/{handleOrDid}/posts`, and synthetic `POST /v1/posts` responses populate the fields. For synthetic post-create responses, counts must be zero and viewer booleans false unless the request is a reply; reply count still zero for the newly-created post. For read/list endpoints, use batch store methods rather than N+1 queries.

  **Must NOT do**: Do not remove or rename existing fields (`reply`, `quote`, `tags`, `author`). Do not include actor arrays.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: focused response-shape change once store methods exist.
  - Skills: [] - Backend Go only.
  - Omitted: [`frontend-ui-ux`] - No UI work.

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: Tasks 5, 7, 8 response assertions | Blocked By: Task 3

  **References** (executor has NO interview context - be exhaustive):
  - Pattern: `appview/internal/api/post_response.go:35` - canonical post response shape.
  - Pattern: `appview/internal/api/post_response.go:51` - response builder.
  - Pattern: `appview/internal/api/post.go:91` - synthetic post-create response flow.
  - Pattern: `appview/internal/api/post.go:314` - list-by-author handler response flow.
  - Test: `appview/internal/api/post_response_test.go` - response builder tests.
  - Test: `appview/internal/api/post_test.go` - handler response tests.

  **Acceptance Criteria** (agent-executable only):
  - [ ] Unit tests confirm JSON keys are camelCase and values are populated.
  - [ ] Handler/list tests assert counts and viewer booleans for authenticated current DID.
  - [ ] Existing post response tests still pass with new fields.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Single post includes engagement
    Tool: Bash
    Steps: Run targeted API handler/store test for GET /v1/posts/{did}/{rkey} with seeded active likes/reposts/replies.
    Expected: Response JSON includes exact likeCount, repostCount, replyCount, viewerHasLiked, viewerHasReposted values.
    Evidence: .sisyphus/evidence/task-4-single-post-engagement.txt

  Scenario: Deleted interaction excluded
    Tool: Bash
    Steps: Run targeted test with one active like and one soft-deleted like on the same post.
    Expected: likeCount is 1 and viewerHasLiked reflects only the active current-viewer row.
    Evidence: .sisyphus/evidence/task-4-deleted-excluded.txt
  ```

  **Commit**: YES | Message: `feat(appview): include engagement on posts` | Files: `appview/internal/api/post_response.go`, `appview/internal/api/post.go`, tests

- [x] 5. Add like/unlike HTTP API

  **What to do**: Add authenticated routes: `POST /v1/posts/{did}/{rkey}/likes` and `DELETE /v1/posts/{did}/{rkey}/likes`. POST accepts an empty body only; reject any non-whitespace request body with `400 unexpected_field`. POST resolves the target post from AppView storage, rejects missing target with `404 not_found`, builds a lexicon body `{"$type":"social.craftsky.feed.like","subject":{"uri":targetURI,"cid":targetCID},"createdAt":serverUTC}` and writes to caller PDS via `CreateRecord`. If the store already has an active like by caller for the subject, return `200 OK` with the active record identity instead of creating a duplicate. If no active row is known, create and return `201 Created`. DELETE finds active caller like for the subject; if found, delete that PDS record by rkey; if none, return `204 No Content`. Both handlers use existing auth/device middleware and error envelope conventions.

  **Must NOT do**: Do not let a caller like an unindexed/missing Craftsky post. Do not create multiple like records when AppView already knows one active row exists. Do not wait for firehose indexing before responding.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: PDS write/delete, idempotency, and auth/error behavior.
  - Skills: [] - Backend Go only.
  - Omitted: [`atproto-lexicon`] - Uses existing lexicon.

  **Parallelization**: Can Parallel: YES | Wave 3 | Blocks: final verification | Blocked By: Tasks 1, 3, 4

  **References** (executor has NO interview context - be exhaustive):
  - Pattern: `appview/internal/routes/routes.go:49` - post route registration with auth/device middleware.
  - Pattern: `appview/internal/api/post.go:25` - PDS-backed create handler structure.
  - Pattern: `appview/internal/api/post.go:81` - `PDSClient.CreateRecord` usage.
  - Pattern: `appview/internal/api/post.go:263` - idempotent delete route behavior.
  - API/Type: `appview/internal/auth/pds_client.go:25` - PDS client interface supports CreateRecord and DeleteRecord.
  - Lexicon: `lexicon/social/craftsky/feed/like.json:3` - like NSID.
  - Lexicon: `lexicon/social/craftsky/feed/like.json:16` - subject strongRef.
  - Test: `appview/internal/api/post_test.go` - handler fake PDS patterns.

  **Acceptance Criteria** (agent-executable only):
  - [ ] Route tests prove unauthenticated/missing device requests are rejected by existing middleware.
  - [ ] Handler tests cover 201 create, 200 already-liked, 204 unlike existing, 204 unlike absent, 404 missing subject, and PDS failure mapping to `502 pds_write_failed` or existing closest error code.
  - [ ] POST body sent to fake PDS has `$type`, `subject.uri`, `subject.cid`, and `createdAt`.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Like post through AppView
    Tool: Bash
    Steps: Run targeted handler test for POST /v1/posts/did:plc:alice/rk1/likes as did:plc:bob with seeded target post.
    Expected: Fake PDS receives collection social.craftsky.feed.like and response status is 201 with subject matching target post URI/CID.
    Evidence: .sisyphus/evidence/task-5-like-create.txt

  Scenario: Unlike is idempotent
    Tool: Bash
    Steps: Run targeted handler tests for DELETE /v1/posts/did:plc:alice/rk1/likes with and without an active like row.
    Expected: Both cases return 204; existing row case calls DeleteRecord with the like rkey, absent case does not call PDS delete.
    Evidence: .sisyphus/evidence/task-5-unlike-idempotent.txt
  ```

  **Commit**: YES | Message: `feat(appview): add like endpoints` | Files: `appview/internal/api/*like*`, `appview/internal/routes/routes.go`, tests

- [x] 6. Add repost/unrepost HTTP API

  **What to do**: Add authenticated routes: `POST /v1/posts/{did}/{rkey}/reposts` and `DELETE /v1/posts/{did}/{rkey}/reposts`. Mirror Task 5 semantics with collection `social.craftsky.feed.repost`: POST accepts an empty body only and rejects any non-whitespace body with `400 unexpected_field`; resolve target post from AppView, reject missing with 404, create `subject` strongRef + server `createdAt` record through PDS, return 201 for newly-created, return 200 with active record identity if already reposted, and make DELETE idempotent with 204. The response shape matches like responses.

  **Must NOT do**: Do not implement quote reposts here. Do not accept request text/body for repost creation; pure repost has no text.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: similar to likes but must avoid quote-repost scope creep.
  - Skills: [] - Backend Go only.
  - Omitted: [`atproto-lexicon`] - Uses existing lexicon.

  **Parallelization**: Can Parallel: YES | Wave 3 | Blocks: final verification | Blocked By: Tasks 1, 3, 4

  **References** (executor has NO interview context - be exhaustive):
  - Pattern: Task 5 like handlers after implemented.
  - Pattern: `appview/internal/api/post.go:263` - idempotent delete route behavior.
  - API/Type: `appview/internal/auth/pds_client.go:25` - PDS client interface supports CreateRecord and DeleteRecord.
  - Lexicon: `lexicon/social/craftsky/feed/repost.json:3` - repost NSID.
  - Lexicon: `lexicon/social/craftsky/feed/repost.json:16` - subject strongRef.
  - Generated: `appview/internal/lexicon/craftsky/feedrepost.go:16` - generated repost type for indexer consistency.

  **Acceptance Criteria** (agent-executable only):
  - [ ] Handler tests cover 201 create, 200 already-reposted, 204 unrepost existing, 204 unrepost absent, 404 missing subject, and PDS failure mapping.
  - [ ] POST rejects non-empty request bodies with `400 unexpected_field`; no text/quote fields accepted.
  - [ ] Route tests prove exact method/path registration.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Repost post through AppView
    Tool: Bash
    Steps: Run targeted handler test for POST /v1/posts/did:plc:alice/rk1/reposts as did:plc:bob with seeded target post.
    Expected: Fake PDS receives collection social.craftsky.feed.repost and response status is 201 with subject matching target post URI/CID.
    Evidence: .sisyphus/evidence/task-6-repost-create.txt

  Scenario: Quote repost not accepted
    Tool: Bash
    Steps: Run targeted handler test sending text or embed payload to POST /v1/posts/{did}/{rkey}/reposts.
    Expected: Handler returns 400 `unexpected_field` and does not call PDS CreateRecord.
    Evidence: .sisyphus/evidence/task-6-no-quote-repost.txt
  ```

  **Commit**: YES | Message: `feat(appview): add repost endpoints` | Files: `appview/internal/api/*repost*`, `appview/internal/routes/routes.go`, tests

- [x] 7. Add direct replies read API

  **What to do**: Add `GET /v1/posts/{did}/{rkey}/replies` returning direct replies whose `reply_parent_uri` equals the target post URI. Use existing post response shape with engagement fields. Support `limit` with default 50 and max 100, and `cursor` using the existing opaque cursor helper. Sort replies oldest-first by `(created_at ASC, uri ASC)` for conversation readability; cursor must preserve this order. Resolve handles using the existing `HandleResolver`; if handle resolution fails for an author, match current post behavior rather than inventing fallback semantics.

  **Must NOT do**: Do not return nested descendants from this endpoint. Do not include actor-list engagement data.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: new paginated read endpoint with post hydration.
  - Skills: [] - Backend Go only.
  - Omitted: [`agent-browser`] - No browser/UI verification.

  **Parallelization**: Can Parallel: YES | Wave 3 | Blocks: final verification | Blocked By: Tasks 1, 3, 4

  **References** (executor has NO interview context - be exhaustive):
  - Storage: `appview/migrations/000010_craftsky_posts.up.sql:34` - `reply_parent_uri` index exists.
  - Pattern: `appview/internal/api/post_store.go:111` - list pagination and cursor style.
  - Pattern: `appview/internal/api/post.go:314` - list handler structure.
  - Pattern: `appview/internal/routes/routes.go:57` - existing list route registration.
  - Response: `appview/internal/api/post_response.go:35` - canonical post response.
  - Test: `appview/internal/api/post_store_test.go` - store test style.
  - Test: `appview/internal/api/post_test.go` - handler test style.

  **Acceptance Criteria** (agent-executable only):
  - [ ] Store tests cover direct replies only, excluding siblings and nested grandchildren.
  - [ ] Handler tests cover default limit, max limit clamping/rejection per existing convention, cursor second page, missing target 404, invalid DID 400.
  - [ ] Response items include engagement fields.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: List direct replies
    Tool: Bash
    Steps: Run targeted handler test seeding one parent, two direct replies, and one grandchild reply.
    Expected: GET /v1/posts/{did}/{rkey}/replies returns exactly the two direct replies in createdAt ascending order.
    Evidence: .sisyphus/evidence/task-7-direct-replies.txt

  Scenario: Replies cursor pagination
    Tool: Bash
    Steps: Run targeted handler/store test requesting limit=1 over two direct replies, then request the returned cursor.
    Expected: First response has one item and non-empty cursor; second response has the second item and empty cursor.
    Evidence: .sisyphus/evidence/task-7-replies-pagination.txt
  ```

  **Commit**: YES | Message: `feat(appview): add reply listing endpoint` | Files: `appview/internal/api/post_store.go`, `appview/internal/api/post.go`, `appview/internal/routes/routes.go`, tests

- [x] 8. Add nested thread read API

  **What to do**: Add `GET /v1/posts/{did}/{rkey}/thread`. Response shape must be deterministic:
  ```json
  {
    "post": { /* PostResponse */ },
    "replies": [ /* ThreadNode */ ],
    "truncated": false
  }
  ```
  where each `ThreadNode` has `{ "post": PostResponse, "replies": [ThreadNode] }`. The endpoint returns the target post as root of the returned tree and descendants below it, not ancestors above it. Load descendants where `reply_root_uri` is the target URI OR `reply_parent_uri` recursively links below the target; if the target itself is a reply, still return descendants under that target only. Cap traversal at depth 6 and total 500 posts; set `truncated: true` if either cap is hit. Sort siblings oldest-first by `(created_at ASC, uri ASC)`. Missing/deleted replies are omitted; children whose parent row is missing should be omitted unless their parent is the target/root loaded in this response.

  **Must NOT do**: Do not make unbounded recursive queries. Do not include parent/ancestor chain in this phase. Do not call PDS to hydrate missing records.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: recursive data shaping and caps need careful tests.
  - Skills: [] - Backend Go only.
  - Omitted: [`ultrabrain`] - Complexity is moderate and well-scoped.

  **Parallelization**: Can Parallel: YES | Wave 3 | Blocks: final verification | Blocked By: Tasks 1, 3, 4

  **References** (executor has NO interview context - be exhaustive):
  - Storage: `appview/migrations/000010_craftsky_posts.up.sql:13` - reply root/parent columns.
  - Storage: `appview/migrations/000010_craftsky_posts.up.sql:34` - reply indexes.
  - Indexer: `appview/internal/index/craftsky_post.go:100` - how reply root/parent pointers are materialized.
  - Response: `appview/internal/api/post_response.go:75` - reply response shape.
  - Pattern: `appview/internal/api/post.go:216` - get single post handler flow around path DID/rkey parsing.
  - Test: `appview/internal/api/post_response_test.go` - response structure tests.

  **Acceptance Criteria** (agent-executable only):
  - [ ] Handler tests cover target with no replies, nested replies to depth 3, cap/truncation behavior, missing target 404, invalid DID 400.
  - [ ] Store/thread builder tests prove sibling ordering is stable.
  - [ ] Response JSON uses camelCase keys and no null `replies` arrays; empty arrays are `[]`.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Nested thread tree
    Tool: Bash
    Steps: Run targeted handler test seeding root -> reply A -> reply B and root -> reply C.
    Expected: GET /v1/posts/{did}/{rkey}/thread returns root post with two top-level replies, and reply A contains reply B as its child.
    Evidence: .sisyphus/evidence/task-8-thread-tree.txt

  Scenario: Thread traversal cap
    Tool: Bash
    Steps: Run targeted store/thread test seeding more than the total-node cap or depth greater than 6.
    Expected: Response sets truncated=true and does not return more than 500 posts or depth greater than 6.
    Evidence: .sisyphus/evidence/task-8-thread-cap.txt
  ```

  **Commit**: YES | Message: `feat(appview): add thread endpoint` | Files: `appview/internal/api/post_store.go`, `appview/internal/api/post_response.go`, `appview/internal/api/post.go`, `appview/internal/routes/routes.go`, tests

## Final Verification Wave (MANDATORY — after ALL implementation tasks)
> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**
> **Never mark F1-F4 as checked before getting user's okay.** Rejection or user feedback -> fix -> re-run -> present again -> wait for okay.
- [x] F1. Plan Compliance Audit — oracle
- [x] F2. Code Quality Review — unspecified-high
- [x] F3. Real Manual QA — unspecified-high (+ playwright if UI; not expected here)
- [x] F4. Scope Fidelity Check — deep

## Commit Strategy
- Prefer one commit after all backend work passes verification: `feat(appview): add backend interactions for posts`.
- If implementation is split by agent waves, use focused conventional commits only after each wave passes its own tests.
- Do not commit `.sisyphus/evidence/` unless the repository convention requires it.

## Success Criteria
- Like/repost records are written to and deleted from the caller's PDS through AppView endpoints.
- Firehose indexing materializes active like/repost state in Postgres without duplicate counts.
- Post responses include engagement counts and viewer state.
- Reply creation remains through `POST /v1/posts`; reply reads and thread reads work from AppView state.
- All new behavior has automated tests and final `just test` passes.
